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

// GradingScale defines the minimum percentage cutoffs for letter grades A through E.
// Any percentage below ScaleE results in grade F (fail).
type GradingScale struct {
	ScaleA float64 `json:"scaleA" gorm:"column:scale_a;not null"`
	ScaleB float64 `json:"scaleB" gorm:"column:scale_b;not null"`
	ScaleC float64 `json:"scaleC" gorm:"column:scale_c;not null"`
	ScaleD float64 `json:"scaleD" gorm:"column:scale_d;not null"`
	ScaleE float64 `json:"scaleE" gorm:"column:scale_e;not null"`
}

// DefaultGradingScale returns standard Norwegian university scale cutoffs:
// A >= 90%, B >= 80%, C >= 60%, D >= 50%, E >= 40%, F < 40%.
func DefaultGradingScale() GradingScale {
	return GradingScale{
		ScaleA: 90,
		ScaleB: 80,
		ScaleC: 60,
		ScaleD: 50,
		ScaleE: 40,
	}
}

// Validate checks that thresholds are strictly descending and within (0, 100].
func (s GradingScale) Validate() error {
	if s.ScaleA > 100 || s.ScaleA <= s.ScaleB ||
		s.ScaleB <= s.ScaleC ||
		s.ScaleC <= s.ScaleD ||
		s.ScaleD <= s.ScaleE ||
		s.ScaleE <= 0 {
		return ErrInvalidGradingScale
	}
	return nil
}

// CalculateGrade maps a percentage (0-100) to a letter grade A-F according to the scale.
func (s GradingScale) CalculateGrade(percent float64) string {
	switch {
	case percent >= s.ScaleA:
		return "A"
	case percent >= s.ScaleB:
		return "B"
	case percent >= s.ScaleC:
		return "C"
	case percent >= s.ScaleD:
		return "D"
	case percent >= s.ScaleE:
		return "E"
	default:
		return "F"
	}
}

// Course is the central entity of the learning management system.
type Course struct {
	gorm.Model
	Uuid         uuid.UUID    `json:"uuid" gorm:"type:uuid;uniqueIndex;not null"`
	Title        string       `json:"title" gorm:"type:varchar(255);not null"`
	Description  string       `json:"description" gorm:"type:text"`
	Term         Term         `json:"term" gorm:"type:varchar(10);not null"`
	Year         int          `json:"year" gorm:"not null"`
	GradingScale GradingScale `json:"gradingScale" gorm:"embedded"`
	Modules      []Module     `json:"modules,omitempty" gorm:"foreignKey:CourseID;constraint:OnDelete:CASCADE;"`
	Groups       []Group      `json:"groups,omitempty" gorm:"foreignKey:CourseID;constraint:OnDelete:CASCADE;"`
	Enrollments  []Enrollment `json:"enrollments,omitempty" gorm:"foreignKey:CourseID;constraint:OnDelete:CASCADE;"`
}

func (c *Course) BeforeCreate(tx *gorm.DB) error {
	if c.Uuid == uuid.Nil {
		c.Uuid = uuid.New()
	}
	if c.GradingScale.ScaleA == 0 && c.GradingScale.ScaleB == 0 && c.GradingScale.ScaleC == 0 && c.GradingScale.ScaleD == 0 && c.GradingScale.ScaleE == 0 {
		c.GradingScale = DefaultGradingScale()
	}
	return nil
}

func (c *Course) Update(updated *Course) *Course {
	c.Title = updated.Title
	c.Description = updated.Description
	c.Term = updated.Term
	c.Year = updated.Year
	if updated.GradingScale != (GradingScale{}) {
		c.GradingScale = updated.GradingScale
	}
	return c
}

func (c *Course) CalculateGrade(percent float64) string {
	return c.GradingScale.CalculateGrade(percent)
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
	Uuid       uuid.UUID   `json:"uuid" gorm:"type:uuid;uniqueIndex;not null"`
	CourseID   uint        `json:"courseId" gorm:"not null;index"`
	Course     *Course     `json:"course,omitempty" gorm:"foreignKey:CourseID;constraint:OnDelete:CASCADE;"`
	Name       string      `json:"name" gorm:"type:varchar(255);not null"`
	MaxMembers int         `json:"maxMembers" gorm:"not null;default:4"`
	Members    []user.User `json:"members,omitempty" gorm:"many2many:group_members;constraint:OnDelete:CASCADE;"`
}

func (g *Group) BeforeCreate(tx *gorm.DB) error {
	if g.Uuid == uuid.Nil {
		g.Uuid = uuid.New()
	}
	if g.MaxMembers <= 0 {
		g.MaxMembers = 4
	}
	return nil
}

// Module represents an ordered structural unit within a course (e.g., "Week 1: Introduction").
// Modules support prerequisite dependencies via DependsOnID.
type Module struct {
	gorm.Model
	Uuid        uuid.UUID `json:"uuid" gorm:"type:uuid;uniqueIndex"`
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
