package course

import (
	"testing"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestCreateCourseDto_ToModel(t *testing.T) {
	t.Run("valid dto converts successfully", func(t *testing.T) {
		dto := CreateCourseDto{
			Title:       "Software Technologies",
			Description: "DAT250 advanced course",
			Term:        "AUTUMN",
			Year:        2026,
			GradingScale: &GradingScaleDto{
				ScaleA: 95,
				ScaleB: 85,
				ScaleC: 70,
				ScaleD: 55,
				ScaleE: 45,
			},
		}
		model, err := dto.ToModel()
		assert.NoError(t, err)
		assert.NotNil(t, model)
		assert.Equal(t, "Software Technologies", model.Title)
		assert.Equal(t, "DAT250 advanced course", model.Description)
		assert.Equal(t, TermAutumn, model.Term)
		assert.Equal(t, 2026, model.Year)
		assert.Equal(t, 95.0, model.GradingScale.ScaleA)
		assert.Equal(t, 45.0, model.GradingScale.ScaleE)
	})

	t.Run("invalid grading scale returns ErrInvalidGradingScale", func(t *testing.T) {
		dto := CreateCourseDto{
			Title: "Test",
			Term:  "AUTUMN",
			Year:  2026,
			GradingScale: &GradingScaleDto{
				ScaleA: 80,
				ScaleB: 85, // invalid: A <= B
				ScaleC: 60,
				ScaleD: 50,
				ScaleE: 40,
			},
		}
		model, err := dto.ToModel()
		assert.ErrorIs(t, err, ErrInvalidGradingScale)
		assert.Nil(t, model)
	})

	t.Run("invalid term returns ErrInvalidTerm", func(t *testing.T) {
		dto := CreateCourseDto{
			Title: "Test",
			Term:  "SUMMER",
			Year:  2026,
		}
		model, err := dto.ToModel()
		assert.ErrorIs(t, err, ErrInvalidTerm)
		assert.Nil(t, model)
	})

	t.Run("invalid year returns ErrInvalidYear", func(t *testing.T) {
		dto := CreateCourseDto{
			Title: "Test",
			Term:  "SPRING",
			Year:  1999,
		}
		model, err := dto.ToModel()
		assert.ErrorIs(t, err, ErrInvalidYear)
		assert.Nil(t, model)
	})
}

func TestUpdateCourseDto_ToModel(t *testing.T) {
	t.Run("valid dto converts successfully", func(t *testing.T) {
		dto := UpdateCourseDto{
			Title:       "Updated Title",
			Description: "Updated Desc",
			Term:        "SPRING",
			Year:        2027,
			GradingScale: &GradingScaleDto{
				ScaleA: 92,
				ScaleB: 82,
				ScaleC: 65,
				ScaleD: 52,
				ScaleE: 42,
			},
		}
		model, err := dto.ToModel()
		assert.NoError(t, err)
		assert.NotNil(t, model)
		assert.Equal(t, "Updated Title", model.Title)
		assert.Equal(t, TermSpring, model.Term)
		assert.Equal(t, 2027, model.Year)
		assert.Equal(t, 92.0, model.GradingScale.ScaleA)
	})

	t.Run("invalid grading scale in update returns ErrInvalidGradingScale", func(t *testing.T) {
		dto := UpdateCourseDto{
			Title: "Test",
			Term:  "SPRING",
			Year:  2027,
			GradingScale: &GradingScaleDto{
				ScaleA: 90,
				ScaleB: 80,
				ScaleC: 60,
				ScaleD: 50,
				ScaleE: -1, // invalid: <= 0
			},
		}
		model, err := dto.ToModel()
		assert.ErrorIs(t, err, ErrInvalidGradingScale)
		assert.Nil(t, model)
	})

	t.Run("omitted grading scale leaves zero GradingScale on UpdateCourseDto", func(t *testing.T) {
		dto := UpdateCourseDto{
			Title: "Partial Update",
			Term:  "SPRING",
			Year:  2027,
		}
		model, err := dto.ToModel()
		assert.NoError(t, err)
		assert.Equal(t, GradingScale{}, model.GradingScale)
	})

	t.Run("invalid term returns ErrInvalidTerm", func(t *testing.T) {
		dto := UpdateCourseDto{
			Title: "Updated",
			Term:  "WINTER",
			Year:  2026,
		}
		model, err := dto.ToModel()
		assert.ErrorIs(t, err, ErrInvalidTerm)
		assert.Nil(t, model)
	})

	t.Run("invalid year returns ErrInvalidYear", func(t *testing.T) {
		dto := UpdateCourseDto{
			Title: "Updated",
			Term:  "SPRING",
			Year:  2105,
		}
		model, err := dto.ToModel()
		assert.ErrorIs(t, err, ErrInvalidYear)
		assert.Nil(t, model)
	})
}

func TestCourseDto_FromModel(t *testing.T) {
	t.Run("nil model returns empty dto", func(t *testing.T) {
		dto := CourseDto{}.FromModel(nil)
		assert.Empty(t, dto.Title)
		assert.Empty(t, dto.Uuid)
	})

	t.Run("valid model converts to dto", func(t *testing.T) {
		now := time.Now()
		c := &Course{
			Model: gorm.Model{
				ID:        12,
				CreatedAt: now,
				UpdatedAt: now,
			},
			Uuid:        uuid.New(),
			Title:       "Learning Systems",
			Description: "LMS details",
			Term:        TermAutumn,
			Year:        2026,
		}
		dto := CourseDto{}.FromModel(c)
		assert.Equal(t, c.Uuid.String(), dto.Uuid)
		assert.Equal(t, "Learning Systems", dto.Title)
		assert.Equal(t, "LMS details", dto.Description)
		assert.Equal(t, "AUTUMN", dto.Term)
		assert.Equal(t, 2026, dto.Year)
	})
}

func TestEnrollmentDto_FromModel(t *testing.T) {
	t.Run("nil model returns empty dto", func(t *testing.T) {
		dto := EnrollmentDto{}.FromModel(nil)
		assert.Empty(t, dto.UserEmail)
	})

	t.Run("valid model converts to dto", func(t *testing.T) {
		now := time.Now()
		courseUuid := uuid.New()
		e := &Enrollment{
			Model:      gorm.Model{ID: 5},
			UserID:     10,
			UserEmail:  "student@hvl.no",
			CourseID:   12,
			Course:     &Course{Uuid: courseUuid},
			Role:       RoleStudent,
			EnrolledAt: now,
		}
		dto := EnrollmentDto{}.FromModel(e)
		assert.Equal(t, "student@hvl.no", dto.UserEmail)
		assert.Equal(t, courseUuid.String(), dto.CourseUuid)
		assert.Equal(t, RoleStudent, dto.Role)
		assert.Equal(t, now, dto.EnrolledAt)
	})
}

func TestModuleDtos(t *testing.T) {
	t.Run("CreateModuleDto with position and dependsOn", func(t *testing.T) {
		pos := 3
		depUuid := uuid.New()
		depUuidStr := depUuid.String()
		dto := CreateModuleDto{
			Title:         "Week 2: Advanced Topics",
			Position:      &pos,
			DependsOnUuid: &depUuidStr,
		}
		model, err := dto.ToModel(10)
		assert.NoError(t, err)
		assert.Equal(t, uint(10), model.CourseID)
		assert.Equal(t, "Week 2: Advanced Topics", model.Title)
		assert.Equal(t, 3, model.Position)
		assert.NotNil(t, model.DependsOn)
		assert.Equal(t, depUuid, model.DependsOn.Uuid)

		// nil position defaults to -1 (unspecified)
		dto2 := CreateModuleDto{Title: "Week 1"}
		model2, err := dto2.ToModel(10)
		assert.NoError(t, err)
		assert.Equal(t, -1, model2.Position)

		// invalid dependsOnUuid returns error
		badDep := "not-a-uuid"
		dtoBad := CreateModuleDto{Title: "Week 1", DependsOnUuid: &badDep}
		_, err = dtoBad.ToModel(10)
		assert.Error(t, err)
	})

	t.Run("UpdateModuleDto to model", func(t *testing.T) {
		pos := 5
		depUuid := uuid.New()
		depUuidStr := depUuid.String()
		dto := UpdateModuleDto{
			Title:         "Week 2 Updated",
			Position:      &pos,
			DependsOnUuid: &depUuidStr,
		}
		model, err := dto.ToModel(10)
		assert.NoError(t, err)
		assert.Equal(t, uint(10), model.CourseID)
		assert.Equal(t, "Week 2 Updated", model.Title)
		assert.Equal(t, 5, model.Position)
		assert.NotNil(t, model.DependsOn)
		assert.Equal(t, depUuid, model.DependsOn.Uuid)

		// nil position defaults to -1
		dto2 := UpdateModuleDto{Title: "Week 1 Updated"}
		model2, err := dto2.ToModel(10)
		assert.NoError(t, err)
		assert.Equal(t, -1, model2.Position)

		// invalid dependsOnUuid returns error
		badDep := "not-a-uuid"
		dtoBad := UpdateModuleDto{Title: "Week 1 Updated", DependsOnUuid: &badDep}
		_, err = dtoBad.ToModel(10)
		assert.Error(t, err)
	})

	t.Run("ModuleDto FromModel with nil model", func(t *testing.T) {
		dto := ModuleDto{}.FromModel(nil)
		assert.Empty(t, dto.Uuid)
	})

	t.Run("ModuleDto FromModel with DependsOn", func(t *testing.T) {
		depID := uint(1)
		depUuid := uuid.New()
		modUuid := uuid.New()
		dep := &Module{
			Model:    gorm.Model{ID: depID},
			Uuid:     depUuid,
			CourseID: 10,
			Title:    "Prerequisite Module",
			Position: 0,
		}
		m := &Module{
			Model:       gorm.Model{ID: 2},
			Uuid:        modUuid,
			CourseID:    10,
			Title:       "Dependent Module",
			Position:    1,
			DependsOnID: &depID,
			DependsOn:   dep,
		}
		dto := ModuleDto{}.FromModel(m)
		assert.Equal(t, modUuid.String(), dto.Uuid)
		assert.Equal(t, "Dependent Module", dto.Title)
		assert.Equal(t, 1, dto.Position)
		assert.NotNil(t, dto.DependsOn)
		assert.Equal(t, depUuid.String(), dto.DependsOn.Uuid)
		assert.Equal(t, "Prerequisite Module", dto.DependsOn.Title)
	})
}

func TestGroupDtos(t *testing.T) {
	t.Run("GroupDto FromModel with nil model", func(t *testing.T) {
		dto := GroupDto{}.FromModel(nil)
		assert.Empty(t, dto.Uuid)
		assert.Empty(t, dto.Name)
		assert.Empty(t, dto.Members)
	})

	t.Run("GroupDto FromModel with populated model", func(t *testing.T) {
		grpUuid := uuid.New()
		courseUuid := uuid.New()
		userUuid := uuid.New()

		g := &Group{
			Uuid:       grpUuid,
			Name:       "Test Team",
			MaxMembers: 5,
			Course:     &Course{Uuid: courseUuid},
			Members: []user.User{
				{
					Uuid:     userUuid,
					Username: "alice",
					Email:    "alice@example.com",
				},
			},
		}

		dto := GroupDto{}.FromModel(g)
		assert.Equal(t, grpUuid.String(), dto.Uuid)
		assert.Equal(t, courseUuid.String(), dto.CourseUuid)
		assert.Equal(t, "Test Team", dto.Name)
		assert.Equal(t, 5, dto.MaxMembers)
		assert.Len(t, dto.Members, 1)
		assert.Equal(t, userUuid.String(), dto.Members[0].Uuid)
		assert.Equal(t, "alice", dto.Members[0].Username)
		assert.Equal(t, "alice@example.com", dto.Members[0].Email)
	})
}
