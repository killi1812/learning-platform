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
	Unenroll(courseUuid uuid.UUID, userUuid uuid.UUID) error
	GetCourseEnrollments(courseUuid uuid.UUID) ([]Enrollment, error)
	GetUserEnrollments(userUuid uuid.UUID) ([]Enrollment, error)
	IsTeacherOrAdmin(courseUuid uuid.UUID, userUuid uuid.UUID, isSystemAdmin bool) (bool, error)

	CreateModule(courseUuid uuid.UUID, m *Module) (*Module, error)
	GetModule(courseUuid uuid.UUID, moduleID uint) (*Module, error)
	ListModules(courseUuid uuid.UUID) ([]Module, error)
	UpdateModule(courseUuid uuid.UUID, moduleID uint, updated *Module) (*Module, error)
	DeleteModule(courseUuid uuid.UUID, moduleID uint) error
	ReorderModules(courseUuid uuid.UUID, moduleIDs []uint) error
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

// CreateCourse inserts a new course and automatically enrolls the creator as TEACHER.
func (s *CourseService) CreateCourse(c *Course, creatorUuid uuid.UUID) (*Course, error) {
	if c.Uuid == uuid.Nil {
		c.Uuid = uuid.New()
	}

	var creator user.User
	if err := s.db.Where("uuid = ?", creatorUuid).First(&creator).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		s.logger.Errorf("Failed to find creator user: %v", err)
		return nil, err
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
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
			User:       &creator,
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
	var existing Course
	if err := s.db.Where("uuid = ?", courseUuid).First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCourseNotFound
		}
		return err
	}

	return s.db.Delete(&existing).Error
}

// SelfEnroll enrolls the authenticated user into the course as a STUDENT.
func (s *CourseService) SelfEnroll(courseUuid uuid.UUID, userUuid uuid.UUID) (*Enrollment, error) {
	var u user.User
	if err := s.db.Where("uuid = ?", userUuid).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	var c Course
	if err := s.db.Where("uuid = ?", courseUuid).First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCourseNotFound
		}
		return nil, err
	}

	var existing Enrollment
	if err := s.db.Where("user_id = ? AND course_id = ?", u.ID, c.ID).First(&existing).Error; err == nil {
		return nil, ErrAlreadyEnrolled
	}

	enrollment := Enrollment{
		UserID:     u.ID,
		UserEmail:  u.Email,
		CourseID:   c.ID,
		Course:     &c,
		Role:       RoleStudent,
		EnrolledAt: time.Now(),
	}

	if err := s.db.Create(&enrollment).Error; err != nil {
		return nil, err
	}
	enrollment.User = &u
	return &enrollment, nil
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

	var c Course
	if err := s.db.Where("uuid = ?", courseUuid).First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCourseNotFound
		}
		return nil, err
	}

	var existing Enrollment
	if err := s.db.Where("user_id = ? AND course_id = ?", u.ID, c.ID).First(&existing).Error; err == nil {
		return nil, ErrAlreadyEnrolled
	}

	enrollment := Enrollment{
		UserID:     u.ID,
		UserEmail:  u.Email,
		CourseID:   c.ID,
		Course:     &c,
		Role:       role,
		EnrolledAt: time.Now(),
	}

	if err := s.db.Create(&enrollment).Error; err != nil {
		return nil, err
	}
	enrollment.User = &u
	return &enrollment, nil
}

// Unenroll removes a user's enrollment from the specified course.
func (s *CourseService) Unenroll(courseUuid uuid.UUID, userUuid uuid.UUID) error {
	var u user.User
	if err := s.db.Where("uuid = ?", userUuid).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	var c Course
	if err := s.db.Where("uuid = ?", courseUuid).First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCourseNotFound
		}
		return err
	}

	res := s.db.Where("user_id = ? AND course_id = ?", u.ID, c.ID).Delete(&Enrollment{})
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
	var c Course
	if err := s.db.Where("uuid = ?", courseUuid).First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCourseNotFound
		}
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
	var u user.User
	if err := s.db.Where("uuid = ?", userUuid).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
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

	var u user.User
	if err := s.db.Where("uuid = ?", userUuid).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, ErrUserNotFound
		}
		return false, err
	}

	var c Course
	if err := s.db.Where("uuid = ?", courseUuid).First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, ErrCourseNotFound
		}
		return false, err
	}

	var enrollment Enrollment
	err := s.db.Where("user_id = ? AND course_id = ? AND role = ?", u.ID, c.ID, RoleTeacher).First(&enrollment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// CreateModule adds a module to the course, validating prerequisites and auto-incrementing position if not specified.
func (s *CourseService) CreateModule(courseUuid uuid.UUID, m *Module) (*Module, error) {
	var c Course
	if err := s.db.Where("uuid = ?", courseUuid).First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCourseNotFound
		}
		return nil, err
	}

	if m.DependsOnID != nil {
		var dep Module
		if err := s.db.First(&dep, *m.DependsOnID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrModuleNotFound
			}
			return nil, err
		}
		if dep.CourseID != c.ID {
			return nil, ErrCrossCoursePrerequisite
		}
	}

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

	return s.GetModule(courseUuid, m.ID)
}

// GetModule fetches a module by ID ensuring it belongs to the given course.
func (s *CourseService) GetModule(courseUuid uuid.UUID, moduleID uint) (*Module, error) {
	var c Course
	if err := s.db.Where("uuid = ?", courseUuid).First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCourseNotFound
		}
		return nil, err
	}

	var m Module
	if err := s.db.Preload("DependsOn").Where("id = ? AND course_id = ?", moduleID, c.ID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrModuleNotFound
		}
		return nil, err
	}
	return &m, nil
}

// ListModules retrieves all modules for a course ordered by position.
func (s *CourseService) ListModules(courseUuid uuid.UUID) ([]Module, error) {
	var c Course
	if err := s.db.Where("uuid = ?", courseUuid).First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCourseNotFound
		}
		return nil, err
	}

	var modules []Module
	if err := s.db.Preload("DependsOn").Where("course_id = ?", c.ID).Order("position asc, id asc").Find(&modules).Error; err != nil {
		return nil, err
	}
	return modules, nil
}

// UpdateModule modifies title, position, and prerequisite of an existing module.
func (s *CourseService) UpdateModule(courseUuid uuid.UUID, moduleID uint, updated *Module) (*Module, error) {
	var c Course
	if err := s.db.Where("uuid = ?", courseUuid).First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCourseNotFound
		}
		return nil, err
	}

	var existing Module
	if err := s.db.Where("id = ? AND course_id = ?", moduleID, c.ID).First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrModuleNotFound
		}
		return nil, err
	}

	if updated.DependsOnID != nil {
		if *updated.DependsOnID == moduleID {
			return nil, ErrSelfPrerequisite
		}
		var dep Module
		if err := s.db.First(&dep, *updated.DependsOnID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrModuleNotFound
			}
			return nil, err
		}
		if dep.CourseID != c.ID {
			return nil, ErrCrossCoursePrerequisite
		}
	}

	existing.Title = updated.Title
	existing.Position = updated.Position
	existing.DependsOnID = updated.DependsOnID

	if err := s.db.Save(&existing).Error; err != nil {
		s.logger.Errorf("Failed to update module: %v", err)
		return nil, err
	}

	return s.GetModule(courseUuid, existing.ID)
}

// DeleteModule removes a module from the course.
func (s *CourseService) DeleteModule(courseUuid uuid.UUID, moduleID uint) error {
	var c Course
	if err := s.db.Where("uuid = ?", courseUuid).First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCourseNotFound
		}
		return err
	}

	var existing Module
	if err := s.db.Where("id = ? AND course_id = ?", moduleID, c.ID).First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrModuleNotFound
		}
		return err
	}

	return s.db.Delete(&existing).Error
}

// ReorderModules updates sequential positions for a list of module IDs within a course.
func (s *CourseService) ReorderModules(courseUuid uuid.UUID, moduleIDs []uint) error {
	var c Course
	if err := s.db.Where("uuid = ?", courseUuid).First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCourseNotFound
		}
		return err
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		for idx, modID := range moduleIDs {
			res := tx.Model(&Module{}).Where("id = ? AND course_id = ?", modID, c.ID).Update("position", idx)
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
