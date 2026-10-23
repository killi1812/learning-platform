package course

import (
	"errors"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrCourseNotFound          = errors.New("course not found")
	ErrAlreadyEnrolled         = errors.New("user is already enrolled in this course")
	ErrEnrollmentNotFound      = errors.New("enrollment not found")
	ErrUserNotFound            = errors.New("user not found")
	ErrUnauthorized            = errors.New("user does not have required permissions for this course")
	ErrModuleNotFound          = errors.New("module not found")
	ErrSelfPrerequisite        = errors.New("module cannot depend on itself")
	ErrCrossCoursePrerequisite = errors.New("prerequisite module belongs to a different course")
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

	CreateModule(courseUuid uuid.UUID, m *Module) (*Module, error)
	GetModule(courseUuid uuid.UUID, moduleUuid uuid.UUID) (*Module, error)
	ListModules(courseUuid uuid.UUID) ([]Module, error)
	UpdateModule(courseUuid uuid.UUID, moduleUuid uuid.UUID, updated *Module) (*Module, error)
	DeleteModule(courseUuid uuid.UUID, moduleUuid uuid.UUID) error
	ReorderModules(courseUuid uuid.UUID, moduleUuids []uuid.UUID) error
}

type CourseService struct {
	db     *gorm.DB
	logger *zap.SugaredLogger
}

func NewCourseService(db *gorm.DB, logger *zap.SugaredLogger) ICourseService {
	return &CourseService{
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
	if m.DependsOn != nil && m.DependsOn.Uuid != uuid.Nil {
		if selfUuid != uuid.Nil && m.DependsOn.Uuid == selfUuid {
			return nil, ErrSelfPrerequisite
		}
		var dep Module
		if err := s.db.Where("uuid = ?", m.DependsOn.Uuid).First(&dep).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrModuleNotFound
			}
			return nil, err
		}
		if dep.CourseID != courseID {
			return nil, ErrCrossCoursePrerequisite
		}
		return &dep.ID, nil
	}
	if m.DependsOnID != nil {
		var dep Module
		if err := s.db.First(&dep, *m.DependsOnID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrModuleNotFound
			}
			return nil, err
		}
		if dep.CourseID != courseID {
			return nil, ErrCrossCoursePrerequisite
		}
		if selfUuid != uuid.Nil && dep.Uuid == selfUuid {
			return nil, ErrSelfPrerequisite
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
	return &existing, nil
}

// DeleteCourse removes a course and cascades deletion of modules and enrollments.
func (s *CourseService) DeleteCourse(courseUuid uuid.UUID) error {
	existing, err := s.findCourse(courseUuid)
	if err != nil {
		return err
	}
	return s.db.Unscoped().Delete(existing).Error
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

	res := s.db.Unscoped().Where("user_id = ? AND course_id = ?", u.ID, c.ID).Delete(&Enrollment{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrEnrollmentNotFound
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

	if m.Position == 0 {
		var count int64
		s.db.Model(&Module{}).Where("course_id = ?", c.ID).Count(&count)
		m.Position = int(count)
	}

	if err := s.db.Create(m).Error; err != nil {
		s.logger.Errorf("Failed to create module: %v", err)
		return nil, err
	}
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
	existing.Position = updated.Position
	existing.DependsOnID = depID

	if err := s.db.Save(&existing).Error; err != nil {
		s.logger.Errorf("Failed to update module: %v", err)
		return nil, err
	}
	return s.GetModule(courseUuid, existing.Uuid)
}

// DeleteModule removes a module from the course.
func (s *CourseService) DeleteModule(courseUuid uuid.UUID, moduleUuid uuid.UUID) error {
	existing, err := s.GetModule(courseUuid, moduleUuid)
	if err != nil {
		return err
	}
	return s.db.Unscoped().Delete(existing).Error
}

// ReorderModules updates sequential positions for a list of module UUIDs within a course.
func (s *CourseService) ReorderModules(courseUuid uuid.UUID, moduleUuids []uuid.UUID) error {
	c, err := s.findCourse(courseUuid)
	if err != nil {
		return err
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
