package content_test

import (
	"testing"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/content"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestContentService_CreateAssignmentAndGet(t *testing.T) {
	db := testutil.NewSqliteTestDb(t,
		&course.Course{},
		&course.Module{},
		&content.Content{},
		&content.StaticPage{},
		&content.Assignment{},
		&content.Quiz{},
		&content.Question{},
		&content.CodingExercise{},
	)

	svc := content.NewContentService(db, zap.NewNop().Sugar())

	c := &course.Course{Title: "DAT250", Term: course.TermAutumn, Year: 2026}
	assert.NoError(t, db.Create(c).Error)

	m := &course.Module{CourseID: c.ID, Title: "Week 1", Position: 1}
	assert.NoError(t, db.Create(m).Error)

	t.Run("CreateAssignment succeeds with valid module", func(t *testing.T) {
		deadline := time.Now().Add(48 * time.Hour)
		cnt := &content.Content{
			Title:    "Exercise 1",
			Position: 1,
		}
		asgn := &content.Assignment{
			Type:                   content.AssignmentHandIn,
			Deadline:               &deadline,
			AcceptsLateSubmissions: false,
		}

		created, err := svc.CreateAssignment(m.Uuid, cnt, asgn)
		assert.NoError(t, err)
		assert.NotNil(t, created)
		assert.NotEqual(t, uuid.Nil, created.Uuid)
		assert.Equal(t, m.ID, created.ModuleID)
		assert.Equal(t, content.TypeAssignment, created.Type)
		assert.Equal(t, created.ID, created.Assignment.ContentID)

		// Verify retrieval
		fetched, err := svc.GetContent(created.Uuid)
		assert.NoError(t, err)
		assert.NotNil(t, fetched)
		assert.Equal(t, created.Title, fetched.Title)
		assert.NotNil(t, fetched.Assignment)
		assert.Equal(t, content.AssignmentHandIn, fetched.Assignment.Type)

		// Verify retrieval by module
		byMod, err := svc.GetAssignmentByModule(m.Uuid)
		assert.NoError(t, err)
		assert.NotNil(t, byMod)
		assert.Equal(t, created.Uuid, byMod.Uuid)
		assert.NotNil(t, byMod.Assignment)
	})

	t.Run("CreateAssignment fails when module not found", func(t *testing.T) {
		_, err := svc.CreateAssignment(uuid.New(), &content.Content{Title: "Orphan"}, &content.Assignment{Type: content.AssignmentHandIn})
		assert.ErrorIs(t, err, content.ErrModuleNotFound)
	})

	t.Run("GetContent fails when content not found", func(t *testing.T) {
		_, err := svc.GetContent(uuid.New())
		assert.ErrorIs(t, err, content.ErrContentNotFound)
	})

	t.Run("GetAssignmentByModule fails when no assignment found", func(t *testing.T) {
		_, err := svc.GetAssignmentByModule(uuid.New())
		assert.ErrorIs(t, err, content.ErrContentNotFound)
	})
}
