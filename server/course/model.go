package course

import (
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CourseRole defines the role of a user in a specific course.
// Roles are assigned per course (a user can be a Student in Course A and a Teacher in Course B).
type CourseRole string

const (
	RoleStudent CourseRole = "STUDENT"
	RoleTeacher CourseRole = "TEACHER"
)

// Term represents the academic semester of a course (e.g. Autumn 2026).
type Term string

const (
	TermSpring Term = "SPRING"
	TermAutumn Term = "AUTUMN"
)

// Course is the central entity of the learning management system.
type Course struct {
	gorm.Model
	Uuid        uuid.UUID    `json:"uuid" gorm:"type:uuid;uniqueIndex;not null"`
	Title       string       `json:"title" gorm:"type:varchar(255);not null"`
	Description string       `json:"description" gorm:"type:text"`
	Term        Term         `json:"term" gorm:"type:varchar(10);not null"`
	Year        int          `json:"year" gorm:"not null"`
	Modules     []Module     `json:"modules,omitempty" gorm:"foreignKey:CourseID;constraint:OnDelete:CASCADE;"`
	Groups      []Group      `json:"groups,omitempty" gorm:"foreignKey:CourseID;constraint:OnDelete:CASCADE;"`
	Enrollments []Enrollment `json:"enrollments,omitempty" gorm:"foreignKey:CourseID;constraint:OnDelete:CASCADE;"`
}

func (c *Course) BeforeCreate(tx *gorm.DB) error {
	if c.Uuid == uuid.Nil {
		c.Uuid = uuid.New()
	}
	return nil
}

func (c *Course) Update(updated *Course) *Course {
	c.Title = updated.Title
	c.Description = updated.Description
	c.Term = updated.Term
	c.Year = updated.Year
	return c
}

// Enrollment records a user's participation in a course with their per-course role.
type Enrollment struct {
	gorm.Model
	UserID     uint       `json:"userId" gorm:"not null;uniqueIndex:idx_user_course;index"`
	User       *user.User `json:"user,omitempty" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE;"`
	UserEmail  string     `json:"userEmail,omitempty" gorm:"type:varchar(255);index"`
	CourseID   uint       `json:"courseId" gorm:"not null;uniqueIndex:idx_user_course;index"`
	Course     *Course    `json:"course,omitempty" gorm:"foreignKey:CourseID;constraint:OnDelete:CASCADE;"`
	Role       CourseRole `json:"role" gorm:"type:varchar(20);not null;default:'STUDENT'"` // Role describes what user represents on a given course
	EnrolledAt time.Time  `json:"enrolledAt" gorm:"not null"`
}

// Group represents a collaboration group of students within a course.
type Group struct {
	gorm.Model
	CourseID uint        `json:"courseId" gorm:"not null;index"`
	Course   *Course     `json:"course,omitempty" gorm:"foreignKey:CourseID;constraint:OnDelete:CASCADE;"`
	Name     string      `json:"name" gorm:"type:varchar(255);not null"`
	Members  []user.User `json:"members,omitempty" gorm:"many2many:group_members;constraint:OnDelete:CASCADE;"`
}

// Module represents an ordered structural unit within a course (e.g., "Week 1: Introduction").
// Modules support prerequisite dependencies via DependsOnID.
type Module struct {
	gorm.Model
	Uuid        uuid.UUID `json:"uuid" gorm:"type:uuid;uniqueIndex;not null"`
	CourseID    uint      `json:"courseId" gorm:"not null;index"`
	Course      *Course   `json:"course,omitempty" gorm:"foreignKey:CourseID;constraint:OnDelete:CASCADE;"`
	Title       string    `json:"title" gorm:"type:varchar(255);not null"`
	Position    int       `json:"position" gorm:"not null;default:0"` // Position refers to a index of a module within a course (e.g. 1st, 2nd, 3rd item)
	DependsOnID *uint     `json:"dependsOnId,omitempty" gorm:"index"`
	DependsOn   *Module   `json:"dependsOn,omitempty" gorm:"foreignKey:DependsOnID;constraint:OnDelete:SET NULL;"`
}

func (m *Module) BeforeCreate(tx *gorm.DB) error {
	if m.Uuid == uuid.Nil {
		m.Uuid = uuid.New()
	}
	return nil
}
