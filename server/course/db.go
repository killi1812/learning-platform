package course

import (
	"errors"
	"fmt"
	"html"
	"strings"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/notification"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/cerror"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/storage"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	DefaultGroupMaxMembers = 4
	MaxGroupCapacity       = 100
	MaxGroupsPerCourse     = 100
	MaxBulkCreateCount     = 50
)

var (
	ErrCourseNotFound           = errors.New("course not found")
	ErrAlreadyEnrolled          = errors.New("user is already enrolled in this course")
	ErrEnrollmentNotFound       = errors.New("enrollment not found")
	ErrCannotRemoveTeacher      = errors.New("cannot remove teacher from course")
	ErrUserNotFound             = errors.New("user not found")
	ErrUnauthorized             = errors.New("user does not have required permissions for this course")
	ErrModuleNotFound           = errors.New("module not found")
	ErrSelfPrerequisite         = errors.New("module cannot depend on itself")
	ErrCrossCoursePrerequisite  = errors.New("prerequisite module belongs to a different course")
	ErrCircularPrerequisite     = errors.New("circular module prerequisite dependency detected")
	ErrInvalidReorderList       = errors.New("reorder list must contain all modules of the course without duplicates")
	ErrGroupNotFound            = errors.New("group not found")
	ErrUserAlreadyInGroup       = errors.New("user is already in a group for this course")
	ErrUserNotInGroup           = errors.New("user is not a member of this group")
	ErrGroupFull                = errors.New("group has reached its maximum member capacity")
	ErrGroupMaxMembersTooSmall  = errors.New("max members cannot be less than current member count")
	ErrCourseGroupLimitExceeded = errors.New("course cannot exceed 100 groups")
	ErrUserAlreadyInGroupError  = ErrUserAlreadyInGroup
)

type ICourseService interface {
	CreateCourse(c *Course, creatorUuid uuid.UUID) (*Course, error)
	GetCourse(courseUuid uuid.UUID) (*Course, error)
	ListCourses(term *Term, year *int) ([]Course, error)
	UpdateCourse(courseUuid uuid.UUID, updated *Course) (*Course, error)
	DeleteCourse(courseUuid uuid.UUID) error

	SelfEnroll(courseUuid uuid.UUID, userUuid uuid.UUID) (*Enrollment, error)
	EnrollByEmail(courseUuid uuid.UUID, email string, role CourseRole) (*Enrollment, error)
	Withdraw(courseUuid uuid.UUID, userUuid uuid.UUID) error
	GetCourseEnrollments(courseUuid uuid.UUID) ([]Enrollment, error)
	GetUserEnrollments(userUuid uuid.UUID) ([]Enrollment, error)
	IsTeacherOrAdmin(courseUuid uuid.UUID, userUuid uuid.UUID, isSystemAdmin bool) (bool, error)
	IsEnrolledOrAdmin(courseUuid uuid.UUID, userUuid uuid.UUID, isSystemAdmin bool) (bool, error)

	CreateModule(courseUuid uuid.UUID, m *Module) (*Module, error)
	GetModule(courseUuid uuid.UUID, moduleUuid uuid.UUID) (*Module, error)
	ListModules(courseUuid uuid.UUID) ([]Module, error)
	UpdateModule(courseUuid uuid.UUID, moduleUuid uuid.UUID, updated *Module) (*Module, error)
	DeleteModule(courseUuid uuid.UUID, moduleUuid uuid.UUID) error
	ReorderModules(courseUuid uuid.UUID, moduleUuids []uuid.UUID) error
}
type CourseService struct {
	db            *gorm.DB
	storage       storage.IStorageService
	notifications notification.INotificationService
	logger        *zap.SugaredLogger
}

type IGroupService interface {
	ListCourseGroups(courseUuid uuid.UUID) ([]Group, error)
	GetGroup(groupUuid uuid.UUID) (*Group, error)
	CreateGroup(courseUuid uuid.UUID, name string, maxMembers int) (*Group, error)
	BulkCreateGroups(courseUuid uuid.UUID, prefix string, count int, maxMembers int) ([]Group, error)
	UpdateGroup(groupUuid uuid.UUID, name string, maxMembers int) (*Group, error)
	DeleteGroup(groupUuid uuid.UUID) error
	UserJoinGroup(groupUuid uuid.UUID, userUuid uuid.UUID) (*Group, error)
	RemoveStudentFromGroup(groupUuid uuid.UUID, userUuid uuid.UUID) (*Group, error)
}
type GroupService struct {
	db     *gorm.DB
	logger *zap.SugaredLogger
}

func NewCourseService(db *gorm.DB, storage storage.IStorageService, notifications notification.INotificationService, logger *zap.SugaredLogger) ICourseService {
	return &CourseService{
		db:            db,
		storage:       storage,
		notifications: notifications,
		logger:        logger,
	}
}

func (s *CourseService) notifyEnrolledStudents(courseID uint, content notification.NotificationContent) {
	var studentUuids []uuid.UUID
	err := s.db.Model(&Enrollment{}).
		Joins("JOIN users ON users.id = enrollments.user_id").
		Where("enrollments.course_id = ? AND enrollments.role = ?", courseID, RoleStudent).
		Pluck("users.uuid", &studentUuids).Error
	if err != nil {
		s.logger.Errorf("Failed to query enrolled students for course %d: %v", courseID, err)
		return
	}
	if len(studentUuids) == 0 {
		return
	}
	if _, err := s.notifications.NotifySystemMany(studentUuids, content); err != nil {
		s.logger.Errorf("Failed to notify enrolled students for course %d: %v", courseID, err)
	}
}

func NewGroupService(db *gorm.DB, logger *zap.SugaredLogger) IGroupService {
	return &GroupService{
		db:     db,
		logger: logger,
	}
}

func (s *CourseService) findCourse(courseUuid uuid.UUID) (*Course, error) {
	var c Course
	if err := s.db.Where("uuid = ?", courseUuid).First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCourseNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (s *CourseService) findUser(userUuid uuid.UUID) (*user.User, error) {
	var u user.User
	if err := s.db.Where("uuid = ?", userUuid).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (s *CourseService) resolvePrerequisite(courseID uint, selfUuid uuid.UUID, m *Module) (*uint, error) {
	var dep Module
	var foundDep bool

	if m.DependsOn != nil && m.DependsOn.Uuid != uuid.Nil {
		if selfUuid != uuid.Nil && m.DependsOn.Uuid == selfUuid {
			return nil, ErrSelfPrerequisite
		}
		if err := s.db.Where("uuid = ?", m.DependsOn.Uuid).First(&dep).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrModuleNotFound
			}
			return nil, err
		}
		foundDep = true
	} else if m.DependsOnID != nil {
		if err := s.db.First(&dep, *m.DependsOnID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrModuleNotFound
			}
			return nil, err
		}
		if selfUuid != uuid.Nil && dep.Uuid == selfUuid {
			return nil, ErrSelfPrerequisite
		}
		foundDep = true
	}

	if foundDep {
		if dep.CourseID != courseID {
			return nil, ErrCrossCoursePrerequisite
		}

		if selfUuid != uuid.Nil {
			var self Module
			if err := s.db.Where("uuid = ? AND course_id = ?", selfUuid, courseID).First(&self).Error; err == nil {
				currDepID := dep.DependsOnID
				visited := map[uint]bool{dep.ID: true}
				for currDepID != nil {
					if *currDepID == self.ID {
						return nil, ErrCircularPrerequisite
					}
					if visited[*currDepID] {
						return nil, ErrCircularPrerequisite
					}
					visited[*currDepID] = true
					var parent Module
					if err := s.db.Select("id, depends_on_id").First(&parent, *currDepID).Error; err != nil {
						break
					}
					currDepID = parent.DependsOnID
				}
			}
		}
		return &dep.ID, nil
	}
	return nil, nil
}

// CreateCourse inserts a new course and automatically enrolls the creator as TEACHER.
func (s *CourseService) CreateCourse(c *Course, creatorUuid uuid.UUID) (*Course, error) {
	creator, err := s.findUser(creatorUuid)
	if err != nil {
		return nil, err
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(c).Error; err != nil {
			return err
		}
		enrollment := Enrollment{
			UserID:     creator.ID,
			UserEmail:  creator.Email,
			CourseID:   c.ID,
			Role:       RoleTeacher,
			EnrolledAt: time.Now(),
		}
		return tx.Create(&enrollment).Error
	})
	if err != nil {
		s.logger.Errorf("Failed to create course with teacher enrollment: %v", err)
		return nil, err
	}

	c.Enrollments = []Enrollment{
		{
			UserID:     creator.ID,
			User:       creator,
			UserEmail:  creator.Email,
			CourseID:   c.ID,
			Role:       RoleTeacher,
			EnrolledAt: time.Now(),
		},
	}
	return c, nil
}

// GetCourse fetches a course by its UUID including its modules and enrollments.
func (s *CourseService) GetCourse(courseUuid uuid.UUID) (*Course, error) {
	var c Course
	if err := s.db.Preload("Modules").Preload("Enrollments.User").Where("uuid = ?", courseUuid).First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCourseNotFound
		}
		return nil, err
	}
	return &c, nil
}

// ListCourses retrieves all courses, optionally filtered by term and/or year.
func (s *CourseService) ListCourses(term *Term, year *int) ([]Course, error) {
	var courses []Course
	query := s.db.Model(&Course{}).Preload("Enrollments.User")
	if term != nil && *term != "" {
		query = query.Where("term = ?", *term)
	}
	if year != nil && *year != 0 {
		query = query.Where("year = ?", *year)
	}
	if err := query.Order("year desc, term asc, title asc").Find(&courses).Error; err != nil {
		return nil, err
	}
	return courses, nil
}

// UpdateCourse modifies title, description, term, or year of an existing course.
func (s *CourseService) UpdateCourse(courseUuid uuid.UUID, updated *Course) (*Course, error) {
	var existing Course
	if err := s.db.Preload("Enrollments.User").Where("uuid = ?", courseUuid).First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCourseNotFound
		}
		return nil, err
	}

	existing.Update(updated)
	if err := s.db.Save(&existing).Error; err != nil {
		return nil, err
	}

	s.notifyEnrolledStudents(existing.ID, notification.NotificationContent{
		Title: fmt.Sprintf("Course Updated: %s", existing.Title),
		Body:  fmt.Sprintf("Course details for %q have been updated.", existing.Title),
		URL:   fmt.Sprintf("/courses/%s", existing.Uuid),
	})

	return &existing, nil
}

// DeleteCourse removes a course and cascades deletion of modules and enrollments.
func (s *CourseService) DeleteCourse(courseUuid uuid.UUID) error {
	existing, err := s.findCourse(courseUuid)
	if err != nil {
		return err
	}

	var storageKeys []string
	_ = s.db.Table("files").
		Joins("JOIN contents ON contents.id = files.content_id").
		Joins("JOIN modules ON modules.id = contents.module_id").
		Where("modules.course_id = ?", existing.ID).
		Pluck("files.storage_key", &storageKeys).Error

	if err := s.db.Unscoped().Delete(existing).Error; err != nil {
		return err
	}

	for _, key := range storageKeys {
		if key != "" {
			_ = s.storage.Delete(key)
		}
	}

	return nil
}

func (s *CourseService) createEnrollment(c *Course, u *user.User, role CourseRole) (*Enrollment, error) {
	var existing Enrollment
	if err := s.db.Where("user_id = ? AND course_id = ?", u.ID, c.ID).First(&existing).Error; err == nil {
		return nil, ErrAlreadyEnrolled
	}

	enrollment := Enrollment{
		UserID:     u.ID,
		User:       u,
		UserEmail:  u.Email,
		CourseID:   c.ID,
		Course:     c,
		Role:       role,
		EnrolledAt: time.Now(),
	}
	if err := s.db.Create(&enrollment).Error; err != nil {
		return nil, err
	}
	return &enrollment, nil
}

// SelfEnroll enrolls the authenticated user into the course as a STUDENT.
func (s *CourseService) SelfEnroll(courseUuid uuid.UUID, userUuid uuid.UUID) (*Enrollment, error) {
	u, err := s.findUser(userUuid)
	if err != nil {
		return nil, err
	}
	c, err := s.findCourse(courseUuid)
	if err != nil {
		return nil, err
	}
	return s.createEnrollment(c, u, RoleStudent)
}

// EnrollByEmail allows a teacher to add a user to the course by email with a specified role.
func (s *CourseService) EnrollByEmail(courseUuid uuid.UUID, email string, role CourseRole) (*Enrollment, error) {
	if role == "" {
		role = RoleStudent
	}
	var u user.User
	if err := s.db.Where("email = ?", email).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	c, err := s.findCourse(courseUuid)
	if err != nil {
		return nil, err
	}
	return s.createEnrollment(c, &u, role)
}

// Withdraw removes a user's enrollment from the specified course.
func (s *CourseService) Withdraw(courseUuid uuid.UUID, userUuid uuid.UUID) error {
	u, err := s.findUser(userUuid)
	if err != nil {
		return err
	}
	c, err := s.findCourse(courseUuid)
	if err != nil {
		return err
	}

	var enrollment Enrollment
	if err := s.db.Where("user_id = ? AND course_id = ?", u.ID, c.ID).First(&enrollment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrEnrollmentNotFound
		}
		return err
	}

	if enrollment.Role == RoleTeacher {
		return ErrCannotRemoveTeacher
	}

	res := s.db.Unscoped().Delete(&enrollment)
	if res.Error != nil {
		return res.Error
	}

	// Remove student from any groups within this course
	var courseGroups []Group
	if err := s.db.Where("course_id = ?", c.ID).Find(&courseGroups).Error; err == nil {
		for _, grp := range courseGroups {
			_ = s.db.Model(&grp).Association("Members").Delete(u)
		}
	}

	return nil
}

// GetCourseEnrollments lists all enrollments for a given course.
func (s *CourseService) GetCourseEnrollments(courseUuid uuid.UUID) ([]Enrollment, error) {
	c, err := s.findCourse(courseUuid)
	if err != nil {
		return nil, err
	}
	var enrollments []Enrollment
	if err := s.db.Preload("Course").Preload("User").Where("course_id = ?", c.ID).Find(&enrollments).Error; err != nil {
		return nil, err
	}
	return enrollments, nil
}

// GetUserEnrollments retrieves all enrollments for a given user, preloading course details.
func (s *CourseService) GetUserEnrollments(userUuid uuid.UUID) ([]Enrollment, error) {
	u, err := s.findUser(userUuid)
	if err != nil {
		return nil, err
	}

	var enrollments []Enrollment
	if err := s.db.Preload("Course.Enrollments.User").Where("user_id = ?", u.ID).Find(&enrollments).Error; err != nil {
		return nil, err
	}
	return enrollments, nil
}

// IsTeacherOrAdmin checks whether the given user is a teacher for the course or a system administrator.
func (s *CourseService) IsTeacherOrAdmin(courseUuid uuid.UUID, userUuid uuid.UUID, isSystemAdmin bool) (bool, error) {
	if isSystemAdmin {
		return true, nil
	}
	u, err := s.findUser(userUuid)
	if err != nil {
		return false, err
	}
	c, err := s.findCourse(courseUuid)
	if err != nil {
		return false, err
	}

	var enrollment Enrollment
	if err := s.db.Where("user_id = ? AND course_id = ? AND role = ?", u.ID, c.ID, RoleTeacher).First(&enrollment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// IsEnrolledOrAdmin checks whether the given user is enrolled in the course or a system administrator.
func (s *CourseService) IsEnrolledOrAdmin(courseUuid uuid.UUID, userUuid uuid.UUID, isSystemAdmin bool) (bool, error) {
	if isSystemAdmin {
		return true, nil
	}
	u, err := s.findUser(userUuid)
	if err != nil {
		return false, err
	}
	c, err := s.findCourse(courseUuid)
	if err != nil {
		return false, err
	}

	var enrollment Enrollment
	if err := s.db.Where("user_id = ? AND course_id = ?", u.ID, c.ID).First(&enrollment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CreateModule adds a module to the course, validating prerequisites and auto-incrementing position if not specified.
func (s *CourseService) CreateModule(courseUuid uuid.UUID, m *Module) (*Module, error) {
	c, err := s.findCourse(courseUuid)
	if err != nil {
		return nil, err
	}

	depID, err := s.resolvePrerequisite(c.ID, uuid.Nil, m)
	if err != nil {
		return nil, err
	}
	m.DependsOnID = depID
	m.DependsOn = nil
	m.CourseID = c.ID

	if m.Position <= 0 {
		var count int64
		s.db.Model(&Module{}).Where("course_id = ?", c.ID).Count(&count)
		if count > 0 {
			var maxPos int
			row := s.db.Model(&Module{}).Where("course_id = ?", c.ID).Select("COALESCE(MAX(position), 0)").Row()
			if err := row.Scan(&maxPos); err != nil {
				return nil, err
			}
			m.Position = maxPos + 1
		} else {
			m.Position = 0
		}
	}

	if err := s.db.Create(m).Error; err != nil {
		s.logger.Errorf("Failed to create module: %v", err)
		return nil, err
	}

	s.notifyEnrolledStudents(c.ID, notification.NotificationContent{
		Title: fmt.Sprintf("Course Updated: %s", c.Title),
		Body:  fmt.Sprintf("A new module %q was added to %q.", m.Title, c.Title),
		URL:   fmt.Sprintf("/courses/%s", c.Uuid),
	})

	return s.GetModule(courseUuid, m.Uuid)
}

// GetModule fetches a module by UUID ensuring it belongs to the given course.
func (s *CourseService) GetModule(courseUuid uuid.UUID, moduleUuid uuid.UUID) (*Module, error) {
	c, err := s.findCourse(courseUuid)
	if err != nil {
		return nil, err
	}

	var m Module
	if err := s.db.Preload("DependsOn").Where("uuid = ? AND course_id = ?", moduleUuid, c.ID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrModuleNotFound
		}
		return nil, err
	}
	return &m, nil
}

// ListModules retrieves all modules for a course ordered by position.
func (s *CourseService) ListModules(courseUuid uuid.UUID) ([]Module, error) {
	c, err := s.findCourse(courseUuid)
	if err != nil {
		return nil, err
	}

	var modules []Module
	if err := s.db.Preload("DependsOn").Where("course_id = ?", c.ID).Order("position asc, id asc").Find(&modules).Error; err != nil {
		return nil, err
	}
	return modules, nil
}

// UpdateModule modifies title, position, and prerequisite of an existing module.
func (s *CourseService) UpdateModule(courseUuid uuid.UUID, moduleUuid uuid.UUID, updated *Module) (*Module, error) {
	c, err := s.findCourse(courseUuid)
	if err != nil {
		return nil, err
	}

	var existing Module
	if err := s.db.Where("uuid = ? AND course_id = ?", moduleUuid, c.ID).First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrModuleNotFound
		}
		return nil, err
	}

	depID, err := s.resolvePrerequisite(c.ID, moduleUuid, updated)
	if err != nil {
		return nil, err
	}

	existing.Title = updated.Title
	if updated.Position >= 0 {
		existing.Position = updated.Position
	}
	existing.DependsOnID = depID

	if err := s.db.Save(&existing).Error; err != nil {
		s.logger.Errorf("Failed to update module: %v", err)
		return nil, err
	}

	s.notifyEnrolledStudents(c.ID, notification.NotificationContent{
		Title: fmt.Sprintf("Course Updated: %s", c.Title),
		Body:  fmt.Sprintf("Module %q in %q has been updated.", existing.Title, c.Title),
		URL:   fmt.Sprintf("/courses/%s", c.Uuid),
	})

	return s.GetModule(courseUuid, existing.Uuid)
}

// DeleteModule removes a module from the course.
func (s *CourseService) DeleteModule(courseUuid uuid.UUID, moduleUuid uuid.UUID) error {
	existing, err := s.GetModule(courseUuid, moduleUuid)
	if err != nil {
		return err
	}

	var storageKeys []string
	_ = s.db.Table("files").
		Joins("JOIN contents ON contents.id = files.content_id").
		Where("contents.module_id = ?", existing.ID).
		Pluck("files.storage_key", &storageKeys).Error

	if err := s.db.Unscoped().Delete(existing).Error; err != nil {
		return err
	}

	for _, key := range storageKeys {
		if key != "" {
			_ = s.storage.Delete(key)
		}
	}

	return nil
}

// ReorderModules updates sequential positions for a list of module UUIDs within a course.
func (s *CourseService) ReorderModules(courseUuid uuid.UUID, moduleUuids []uuid.UUID) error {
	c, err := s.findCourse(courseUuid)
	if err != nil {
		return err
	}

	var existingModules []Module
	if err := s.db.Where("course_id = ?", c.ID).Find(&existingModules).Error; err != nil {
		return err
	}
	if len(moduleUuids) != len(existingModules) {
		return ErrInvalidReorderList
	}
	existingMap := make(map[uuid.UUID]bool, len(existingModules))
	for _, em := range existingModules {
		existingMap[em.Uuid] = true
	}
	seen := make(map[uuid.UUID]bool, len(moduleUuids))
	for _, u := range moduleUuids {
		if !existingMap[u] || seen[u] {
			return ErrInvalidReorderList
		}
		seen[u] = true
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		for idx, modUuid := range moduleUuids {
			res := tx.Model(&Module{}).Where("uuid = ? AND course_id = ?", modUuid, c.ID).Update("position", idx)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return ErrModuleNotFound
			}
		}
		return nil
	})
}

func (g *GroupService) findUser(userUuid uuid.UUID) (*user.User, error) {
	var u user.User
	if err := g.db.Where("uuid = ?", userUuid).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (g *GroupService) findCourse(courseUuid uuid.UUID) (*Course, error) {
	var c Course
	if err := g.db.Where("uuid = ?", courseUuid).First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCourseNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (g *GroupService) ListCourseGroups(courseUuid uuid.UUID) ([]Group, error) {
	c, err := g.findCourse(courseUuid)
	if err != nil {
		return nil, err
	}

	var groups []Group
	if err := g.db.Preload("Course").Preload("Members").
		Where("course_id = ?", c.ID).
		Order("id asc").
		Find(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

func (g *GroupService) GetGroup(groupUuid uuid.UUID) (*Group, error) {
	var group Group
	if err := g.db.Preload("Course").Preload("Members").Where("uuid = ?", groupUuid).First(&group).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGroupNotFound
		}
		return nil, err
	}
	return &group, nil
}

func (g *GroupService) CreateGroup(courseUuid uuid.UUID, name string, maxMembers int) (*Group, error) {
	c, err := g.findCourse(courseUuid)
	if err != nil {
		return nil, err
	}

	trimmedName := html.EscapeString(strings.TrimSpace(name))
	if trimmedName == "" {
		return nil, errors.New("group name cannot be empty")
	}

	var existingCount int64
	if err := g.db.Model(&Group{}).Where("course_id = ?", c.ID).Count(&existingCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count existing groups: %w", err)
	}
	if existingCount >= MaxGroupsPerCourse {
		return nil, ErrCourseGroupLimitExceeded
	}

	var count int64
	if err := g.db.Model(&Group{}).
		Where("course_id = ? AND LOWER(name) = LOWER(?)", c.ID, trimmedName).
		Count(&count).Error; err != nil {
		return nil, fmt.Errorf("failed to check duplicate group name: %w", err)
	}
	if count > 0 {
		return nil, cerror.ErrGroupNameAlreadyTaken
	}

	if maxMembers <= 0 {
		maxMembers = DefaultGroupMaxMembers
	} else if maxMembers > MaxGroupCapacity {
		maxMembers = MaxGroupCapacity
	}

	group := Group{
		Uuid:       uuid.New(),
		CourseID:   c.ID,
		Course:     c,
		Name:       trimmedName,
		MaxMembers: maxMembers,
		Members:    []user.User{},
	}

	if err := g.db.Create(&group).Error; err != nil {
		return nil, fmt.Errorf("failed to create group: %w", err)
	}
	return &group, nil
}

func (g *GroupService) BulkCreateGroups(courseUuid uuid.UUID, prefix string, count int, maxMembers int) ([]Group, error) {
	c, err := g.findCourse(courseUuid)
	if err != nil {
		return nil, err
	}

	prefix = html.EscapeString(strings.TrimSpace(prefix))
	if prefix == "" {
		prefix = "Group"
	}
	if count <= 0 {
		return nil, errors.New("count must be greater than zero")
	}
	if count > MaxBulkCreateCount {
		return nil, fmt.Errorf("count cannot exceed %d", MaxBulkCreateCount)
	}

	var existingCount int64
	if err := g.db.Model(&Group{}).Where("course_id = ?", c.ID).Count(&existingCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count existing groups: %w", err)
	}
	if existingCount+int64(count) > MaxGroupsPerCourse {
		return nil, ErrCourseGroupLimitExceeded
	}

	if maxMembers <= 0 {
		maxMembers = DefaultGroupMaxMembers
	} else if maxMembers > MaxGroupCapacity {
		maxMembers = MaxGroupCapacity
	}

	var existingNames []string
	if err := g.db.Model(&Group{}).Where("course_id = ?", c.ID).Pluck("name", &existingNames).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch existing group names: %w", err)
	}
	existingMap := make(map[string]bool, len(existingNames))
	for _, n := range existingNames {
		existingMap[strings.ToLower(strings.TrimSpace(n))] = true
	}

	newGroups := make([]Group, 0, count)
	candidateNum := 1
	for len(newGroups) < count {
		candidateName := fmt.Sprintf("%s %d", prefix, candidateNum)
		candidateNum++
		if existingMap[strings.ToLower(candidateName)] {
			continue
		}
		existingMap[strings.ToLower(candidateName)] = true

		newGroups = append(newGroups, Group{
			Uuid:       uuid.New(),
			CourseID:   c.ID,
			Course:     c,
			Name:       candidateName,
			MaxMembers: maxMembers,
			Members:    []user.User{},
		})
	}

	err = g.db.Transaction(func(tx *gorm.DB) error {
		for i := range newGroups {
			if err := tx.Create(&newGroups[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to bulk create groups: %w", err)
	}

	return newGroups, nil
}

func (g *GroupService) UpdateGroup(groupUuid uuid.UUID, name string, maxMembers int) (*Group, error) {
	trimmedName := html.EscapeString(strings.TrimSpace(name))
	if trimmedName == "" {
		return nil, errors.New("group name cannot be empty")
	}

	if maxMembers <= 0 {
		maxMembers = DefaultGroupMaxMembers
	} else if maxMembers > MaxGroupCapacity {
		maxMembers = MaxGroupCapacity
	}

	err := g.db.Transaction(func(tx *gorm.DB) error {
		var group Group
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Preload("Members").
			Where("uuid = ?", groupUuid).
			First(&group).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrGroupNotFound
			}
			return fmt.Errorf("failed to fetch group: %w", err)
		}

		if maxMembers < len(group.Members) {
			return ErrGroupMaxMembersTooSmall
		}

		var count int64
		if err := tx.Model(&Group{}).
			Where("course_id = ? AND id != ? AND LOWER(name) = LOWER(?)", group.CourseID, group.ID, trimmedName).
			Count(&count).Error; err != nil {
			return fmt.Errorf("failed to check duplicate group name: %w", err)
		}
		if count > 0 {
			return cerror.ErrGroupNameAlreadyTaken
		}

		group.Name = trimmedName
		group.MaxMembers = maxMembers

		if err := tx.Save(&group).Error; err != nil {
			return fmt.Errorf("failed to save group: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return g.GetGroup(groupUuid)
}

func (g *GroupService) DeleteGroup(groupUuid uuid.UUID) error {
	group, err := g.GetGroup(groupUuid)
	if err != nil {
		return err
	}

	return g.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(group).Association("Members").Clear(); err != nil {
			return fmt.Errorf("failed to clear group members: %w", err)
		}
		if err := tx.Delete(group).Error; err != nil {
			return fmt.Errorf("failed to delete group: %w", err)
		}
		return nil
	})
}

func (g *GroupService) UserJoinGroup(groupUuid uuid.UUID, userUuid uuid.UUID) (*Group, error) {
	u, err := g.findUser(userUuid)
	if err != nil {
		return nil, err
	}

	err = g.db.Transaction(func(tx *gorm.DB) error {
		var group Group
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Preload("Members").
			Where("uuid = ?", groupUuid).
			First(&group).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrGroupNotFound
			}
			return fmt.Errorf("failed to load group: %w", err)
		}

		var enrollment Enrollment
		if err := tx.Where("course_id = ? AND user_id = ? AND role = ?", group.CourseID, u.ID, RoleStudent).First(&enrollment).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrEnrollmentNotFound
			}
			return fmt.Errorf("failed to verify enrollment: %w", err)
		}

		var inAnyGroupCount int64
		err := tx.Table("group_members").
			Joins("JOIN groups ON groups.id = group_members.group_id").
			Where("groups.course_id = ? AND group_members.user_id = ?", group.CourseID, u.ID).
			Count(&inAnyGroupCount).Error
		if err != nil {
			return fmt.Errorf("failed to check existing group membership: %w", err)
		}
		if inAnyGroupCount > 0 {
			return ErrUserAlreadyInGroup
		}

		if len(group.Members) >= group.MaxMembers {
			return ErrGroupFull
		}

		if err := tx.Model(&group).Association("Members").Append(u); err != nil {
			return fmt.Errorf("failed to append member: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return g.GetGroup(groupUuid)
}

func (g *GroupService) RemoveStudentFromGroup(groupUuid uuid.UUID, userUuid uuid.UUID) (*Group, error) {
	u, err := g.findUser(userUuid)
	if err != nil {
		return nil, err
	}

	err = g.db.Transaction(func(tx *gorm.DB) error {
		var group Group
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Preload("Members").
			Where("uuid = ?", groupUuid).
			First(&group).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrGroupNotFound
			}
			return fmt.Errorf("failed to load group: %w", err)
		}

		found := false
		for _, m := range group.Members {
			if m.ID == u.ID {
				found = true
				break
			}
		}
		if !found {
			return ErrUserNotInGroup
		}

		if err := tx.Model(&group).Association("Members").Delete(u); err != nil {
			return fmt.Errorf("failed to delete member: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return g.GetGroup(groupUuid)
}
