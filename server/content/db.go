package content

import (
	"errors"
	"fmt"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrModuleNotFound  = errors.New("module not found")
	ErrContentNotFound = errors.New("content not found")
)

// IContentService defines operations for managing educational content and assignments.
type IContentService interface {
	CreateAssignment(moduleUuid uuid.UUID, cnt *Content, asgn *Assignment) (*Content, error)
	GetContent(contentUuid uuid.UUID) (*Content, error)
	GetAssignmentByModule(moduleUuid uuid.UUID) (*Content, error)
}

// ContentService implements IContentService using GORM.
type ContentService struct {
	db     *gorm.DB
	logger *zap.SugaredLogger
}

// NewContentService constructs a new ContentService.
func NewContentService(db *gorm.DB, logger *zap.SugaredLogger) IContentService {
	return &ContentService{
		db:     db,
		logger: logger,
	}
}

// CreateAssignment persists a new Content record of type ASSIGNMENT and its corresponding Assignment record within the specified module.
func (s *ContentService) CreateAssignment(moduleUuid uuid.UUID, cnt *Content, asgn *Assignment) (*Content, error) {
	var m course.Module
	if err := s.db.Where("uuid = ?", moduleUuid).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrModuleNotFound
		}
		return nil, err
	}

	cnt.ModuleID = m.ID
	cnt.Type = TypeAssignment

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(cnt).Error; err != nil {
			return fmt.Errorf("failed to create content: %w", err)
		}

		asgn.ContentID = cnt.ID
		if err := tx.Create(asgn).Error; err != nil {
			return fmt.Errorf("failed to create assignment: %w", err)
		}

		cnt.Assignment = asgn
		return nil
	})

	if err != nil {
		s.logger.Errorf("Failed to create assignment content: %v", err)
		return nil, err
	}

	return cnt, nil
}

// GetContent retrieves a content item along with its assignment details by UUID.
func (s *ContentService) GetContent(contentUuid uuid.UUID) (*Content, error) {
	var cnt Content
	err := s.db.
		Preload("Assignment.Quiz.Questions").
		Preload("Assignment.CodingExercise").
		Preload("StaticPage").
		Where("uuid = ?", contentUuid).
		First(&cnt).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContentNotFound
		}
		return nil, err
	}

	return &cnt, nil
}

// GetAssignmentByModule retrieves the first assignment content within the specified module.
func (s *ContentService) GetAssignmentByModule(moduleUuid uuid.UUID) (*Content, error) {
	var cnt Content
	err := s.db.
		Joins("JOIN modules ON modules.id = contents.module_id").
		Preload("Assignment").
		Where("modules.uuid = ? AND contents.type = ?", moduleUuid, TypeAssignment).
		First(&cnt).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContentNotFound
		}
		return nil, err
	}

	return &cnt, nil
}
