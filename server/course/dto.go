package course

import (
	"errors"
	"time"
)

var (
	ErrInvalidTerm = errors.New("term must be SPRING or AUTUMN")
	ErrInvalidYear = errors.New("year must be between 2000 and 2100")
	ErrInvalidRole = errors.New("role must be STUDENT or TEACHER")
)

type CreateCourseDto struct {
	Title       string `json:"title" binding:"required,min=2,max=255"`
	Description string `json:"description"`
	Term        string `json:"term" binding:"required,oneof=SPRING AUTUMN"`
	Year        int    `json:"year" binding:"required,min=2000,max=2100"`
}

func (d CreateCourseDto) ToModel() (*Course, error) {
	if d.Term != string(TermSpring) && d.Term != string(TermAutumn) {
		return nil, ErrInvalidTerm
	}
	if d.Year < 2000 || d.Year > 2100 {
		return nil, ErrInvalidYear
	}
	return &Course{
		Title:       d.Title,
		Description: d.Description,
		Term:        Term(d.Term),
		Year:        d.Year,
	}, nil
}

type UpdateCourseDto struct {
	Title       string `json:"title" binding:"required,min=2,max=255"`
	Description string `json:"description"`
	Term        string `json:"term" binding:"required,oneof=SPRING AUTUMN"`
	Year        int    `json:"year" binding:"required,min=2000,max=2100"`
}

func (d UpdateCourseDto) ToModel() (*Course, error) {
	if d.Term != string(TermSpring) && d.Term != string(TermAutumn) {
		return nil, ErrInvalidTerm
	}
	if d.Year < 2000 || d.Year > 2100 {
		return nil, ErrInvalidYear
	}
	return &Course{
		Title:       d.Title,
		Description: d.Description,
		Term:        Term(d.Term),
		Year:        d.Year,
	}, nil
}

type CourseDto struct {
	ID          uint      `json:"id,omitempty"`
	Uuid        string    `json:"uuid"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Term        string    `json:"term"`
	Year        int       `json:"year"`
	Teachers    []string  `json:"teachers"`
}

func (CourseDto) FromModel(c *Course) CourseDto {
	if c == nil {
		return CourseDto{}
	}
	teachers := make([]string, 0)
	for _, e := range c.Enrollments {
		if e.Role == RoleTeacher {
			if e.User != nil && e.User.Username != "" {
				teachers = append(teachers, e.User.Username)
			} else if e.UserEmail != "" {
				teachers = append(teachers, e.UserEmail)
			}
		}
	}
	return CourseDto{
		ID:          c.ID,
		Uuid:        c.Uuid.String(),
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
		Title:       c.Title,
		Description: c.Description,
		Term:        string(c.Term),
		Year:        c.Year,
		Teachers:    teachers,
	}
}

type EnrollmentDto struct {
	ID         uint       `json:"id"`
	UserID     uint       `json:"userId"`
	Username   string     `json:"username,omitempty"`
	UserEmail  string     `json:"userEmail"`
	CourseID   uint       `json:"courseId"`
	CourseUuid string     `json:"courseUuid,omitempty"`
	Role       CourseRole `json:"role"`
	EnrolledAt time.Time  `json:"enrolledAt"`
}

func (EnrollmentDto) FromModel(e *Enrollment) EnrollmentDto {
	if e == nil {
		return EnrollmentDto{}
	}
	dto := EnrollmentDto{
		ID:         e.ID,
		UserID:     e.UserID,
		UserEmail:  e.UserEmail,
		CourseID:   e.CourseID,
		Role:       e.Role,
		EnrolledAt: e.EnrolledAt,
	}
	if e.User != nil {
		dto.Username = e.User.Username
	}
	if e.Course != nil {
		dto.CourseUuid = e.Course.Uuid.String()
	}
	return dto
}

type AddEnrollmentDto struct {
	Email string     `json:"email" binding:"required,email"`
	Role  CourseRole `json:"role" binding:"omitempty,oneof=STUDENT TEACHER"`
}

type MyCourseDto struct {
	Course     CourseDto  `json:"course"`
	Role       CourseRole `json:"role"`
	EnrolledAt time.Time  `json:"enrolledAt"`
}

type CreateModuleDto struct {
	Title       string `json:"title" binding:"required,min=2,max=255"`
	Position    *int   `json:"position,omitempty"`
	DependsOnID *uint  `json:"dependsOnId,omitempty"`
}

func (d CreateModuleDto) ToModel(courseIDs ...uint) *Module {
	pos := 0
	if d.Position != nil {
		pos = *d.Position
	}
	var courseID uint
	if len(courseIDs) > 0 {
		courseID = courseIDs[0]
	}
	return &Module{
		CourseID:    courseID,
		Title:       d.Title,
		Position:    pos,
		DependsOnID: d.DependsOnID,
	}
}

type UpdateModuleDto struct {
	Title       string `json:"title" binding:"required,min=2,max=255"`
	Position    *int   `json:"position,omitempty"`
	DependsOnID *uint  `json:"dependsOnId,omitempty"`
}

func (d UpdateModuleDto) ToModel(courseIDs ...uint) *Module {
	pos := 0
	if d.Position != nil {
		pos = *d.Position
	}
	var courseID uint
	if len(courseIDs) > 0 {
		courseID = courseIDs[0]
	}
	return &Module{
		CourseID:    courseID,
		Title:       d.Title,
		Position:    pos,
		DependsOnID: d.DependsOnID,
	}
}

type ModuleDto struct {
	ID          uint       `json:"id"`
	CourseID    uint       `json:"courseId"`
	Title       string     `json:"title"`
	Position    int        `json:"position"`
	DependsOnID *uint      `json:"dependsOnId,omitempty"`
	DependsOn   *ModuleDto `json:"dependsOn,omitempty"`
}

func (ModuleDto) FromModel(m *Module) ModuleDto {
	if m == nil {
		return ModuleDto{}
	}
	dto := ModuleDto{
		ID:          m.ID,
		CourseID:    m.CourseID,
		Title:       m.Title,
		Position:    m.Position,
		DependsOnID: m.DependsOnID,
	}
	if m.DependsOn != nil {
		depDto := ModuleDto{}.FromModel(m.DependsOn)
		dto.DependsOn = &depDto
	}
	return dto
}

type ReorderModulesDto struct {
	ModuleIDs []uint `json:"moduleIds" binding:"required,min=1"`
}
