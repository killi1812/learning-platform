package course

import (
	"errors"
	"time"

	"github.com/google/uuid"
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
	UserUuid   string     `json:"userUuid,omitempty"`
	Username   string     `json:"username,omitempty"`
	UserEmail  string     `json:"userEmail"`
	CourseUuid string     `json:"courseUuid,omitempty"`
	Role       CourseRole `json:"role"`
	EnrolledAt time.Time  `json:"enrolledAt"`
}

func (EnrollmentDto) FromModel(e *Enrollment) EnrollmentDto {
	if e == nil {
		return EnrollmentDto{}
	}
	dto := EnrollmentDto{
		UserEmail:  e.UserEmail,
		Role:       e.Role,
		EnrolledAt: e.EnrolledAt,
	}
	if e.User != nil {
		dto.UserUuid = e.User.Uuid.String()
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
	Title         string  `json:"title" binding:"required,min=2,max=255"`
	Position      *int    `json:"position,omitempty"`
	DependsOnUuid *string `json:"dependsOnUuid,omitempty"`
}

func (d CreateModuleDto) ToModel(courseIDs ...uint) (*Module, error) {
	var dep *Module
	if d.DependsOnUuid != nil && *d.DependsOnUuid != "" {
		depUuid, err := uuid.Parse(*d.DependsOnUuid)
		if err != nil {
			return nil, errors.New("invalid prerequisite module UUID")
		}
		dep = &Module{Uuid: depUuid}
	}
	pos := -1
	if d.Position != nil {
		pos = *d.Position
	}
	var courseID uint
	if len(courseIDs) > 0 {
		courseID = courseIDs[0]
	}
	return &Module{
		CourseID:  courseID,
		Title:     d.Title,
		Position:  pos,
		DependsOn: dep,
	}, nil
}

type UpdateModuleDto struct {
	Title         string  `json:"title" binding:"required,min=2,max=255"`
	Position      *int    `json:"position,omitempty"`
	DependsOnUuid *string `json:"dependsOnUuid,omitempty"`
}

func (d UpdateModuleDto) ToModel(courseIDs ...uint) (*Module, error) {
	var dep *Module
	if d.DependsOnUuid != nil && *d.DependsOnUuid != "" {
		depUuid, err := uuid.Parse(*d.DependsOnUuid)
		if err != nil {
			return nil, errors.New("invalid prerequisite module UUID")
		}
		dep = &Module{Uuid: depUuid}
	}
	pos := -1
	if d.Position != nil {
		pos = *d.Position
	}
	var courseID uint
	if len(courseIDs) > 0 {
		courseID = courseIDs[0]
	}
	return &Module{
		CourseID:  courseID,
		Title:     d.Title,
		Position:  pos,
		DependsOn: dep,
	}, nil
}

type ModuleDto struct {
	Uuid          string     `json:"uuid"`
	CourseUuid    string     `json:"courseUuid,omitempty"`
	Title         string     `json:"title"`
	Position      int        `json:"position"`
	DependsOnUuid *string    `json:"dependsOnUuid,omitempty"`
	DependsOn     *ModuleDto `json:"dependsOn,omitempty"`
}

func (ModuleDto) FromModel(m *Module) ModuleDto {
	if m == nil {
		return ModuleDto{}
	}
	dto := ModuleDto{
		Uuid:     m.Uuid.String(),
		Title:    m.Title,
		Position: m.Position,
	}
	if m.Course != nil && m.Course.Uuid != uuid.Nil {
		dto.CourseUuid = m.Course.Uuid.String()
	}
	if m.DependsOn != nil {
		depUuid := m.DependsOn.Uuid.String()
		dto.DependsOnUuid = &depUuid
		depDto := ModuleDto{}.FromModel(m.DependsOn)
		dto.DependsOn = &depDto
	}
	return dto
}

type ReorderModulesDto struct {
	ModuleUuids []uuid.UUID `json:"moduleUuids" binding:"required,min=1"`
}

type CreateGroupDto struct {
	CourseUuid string `json:"courseUuid" binding:"required"`
	Name       string `json:"name" binding:"required,min=2,max=255"`
	MaxMembers int    `json:"maxMembers" binding:"required,min=1,max=100"`
}

type BulkCreateGroupsDto struct {
	CourseUuid string `json:"courseUuid" binding:"required"`
	Prefix     string `json:"prefix" binding:"omitempty,max=200"`
	Count      int    `json:"count" binding:"required,min=1,max=50"`
	MaxMembers int    `json:"maxMembers" binding:"required,min=1,max=100"`
}

type UpdateGroupDto struct {
	Name       string `json:"name" binding:"required,min=2,max=255"`
	MaxMembers int    `json:"maxMembers" binding:"required,min=1,max=100"`
}

type GroupMemberDto struct {
	Uuid     string `json:"uuid"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type GroupDto struct {
	Uuid       string           `json:"uuid"`
	CourseUuid string           `json:"courseUuid"`
	Name       string           `json:"name"`
	MaxMembers int              `json:"maxMembers"`
	Members    []GroupMemberDto `json:"members"`
}

func (GroupDto) FromModel(m *Group) GroupDto {
	if m == nil {
		return GroupDto{}
	}
	courseUuid := ""
	if m.Course != nil && m.Course.Uuid != uuid.Nil {
		courseUuid = m.Course.Uuid.String()
	}
	members := make([]GroupMemberDto, 0, len(m.Members))
	for _, u := range m.Members {
		members = append(members, GroupMemberDto{
			Uuid:     u.Uuid.String(),
			Username: u.Username,
			Email:    u.Email,
		})
	}
	return GroupDto{
		Uuid:       m.Uuid.String(),
		CourseUuid: courseUuid,
		Name:       m.Name,
		MaxMembers: m.MaxMembers,
		Members:    members,
	}
}
