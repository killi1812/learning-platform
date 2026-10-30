package content

import (
	"testing"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestFileDto_FromModel(t *testing.T) {
	t.Run("nil model returns empty dto", func(t *testing.T) {
		dto := FileDto{}.FromModel(nil)
		assert.Empty(t, dto.Uuid)
		assert.Empty(t, dto.FileName)
	})

	t.Run("valid model with content and module converts successfully", func(t *testing.T) {
		contentUuid := uuid.New()
		moduleUuid := uuid.New()
		now := time.Now()

		f := &File{
			ContentID:   10,
			FileName:    "lecture1.pptx",
			FileSize:    2048,
			ContentType: "application/vnd.openxmlformats-officedocument.presentationml.presentation",
			StorageKey:  "content/123/lecture1.pptx",
			RichText:    "Week 1 Slides",
			Content: &Content{
				Model: gorm.Model{
					ID:        10,
					CreatedAt: now,
					UpdatedAt: now,
				},
				Uuid:     contentUuid,
				Title:    "Lecture 1: Architecture",
				Position: 0,
				Type:     TypeFile,
				Module: &course.Module{
					Model: gorm.Model{ID: 2},
					Uuid:  moduleUuid,
					Title: "Week 1",
				},
			},
		}

		dto := FileDto{}.FromModel(f)
		assert.Equal(t, contentUuid.String(), dto.Uuid)
		assert.Equal(t, moduleUuid.String(), dto.ModuleUuid)
		assert.Equal(t, "Lecture 1: Architecture", dto.Title)
		assert.Equal(t, "lecture1.pptx", dto.FileName)
		assert.Equal(t, int64(2048), dto.FileSize)
		assert.Equal(t, "application/vnd.openxmlformats-officedocument.presentationml.presentation", dto.ContentType)
		assert.Equal(t, "Week 1 Slides", dto.Description)
		assert.Equal(t, now, dto.CreatedAt)
	})
}
