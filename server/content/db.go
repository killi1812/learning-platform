package content

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/storage"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrContentNotFound     = errors.New("content not found")
	ErrModuleNotFound      = errors.New("module not found")
	ErrCourseNotFound      = errors.New("course not found")
	ErrModuleLocked        = errors.New("prerequisite module is not completed")
	ErrContentNotAvailable = errors.New("content is not currently visible")
)

type IContentService interface {
	UploadFile(courseUuid, moduleUuid uuid.UUID, title, fileName string, fileSize int64, contentType string, fileReader io.Reader, description string, sourceUrl string) (*File, error)
	GetFile(courseUuid, moduleUuid, contentUuid, userUuid uuid.UUID, isPrivileged bool) (*File, error)
	ListFiles(courseUuid, moduleUuid, userUuid uuid.UUID, isPrivileged bool) ([]File, error)
	GetFileStream(courseUuid, moduleUuid, contentUuid, userUuid uuid.UUID, isPrivileged bool) (io.ReadCloser, *File, error)
	UpdateFile(courseUuid, moduleUuid, contentUuid uuid.UUID, title, description string) (*File, error)
	DeleteFile(courseUuid, moduleUuid, contentUuid uuid.UUID) error
	CreateAssignment(moduleUuid uuid.UUID, cnt *Content, asgn *Assignment) (*Content, error)
	GetContent(contentUuid uuid.UUID) (*Content, error)
	GetAssignmentByModule(moduleUuid uuid.UUID) (*Content, error)
}

type ContentService struct {
	db      *gorm.DB
	storage storage.IStorageService
	logger  *zap.SugaredLogger
}

func NewContentService(db *gorm.DB, storage storage.IStorageService, logger *zap.SugaredLogger) IContentService {
	return &ContentService{
		db:      db,
		storage: storage,
		logger:  logger,
	}
}

func (s *ContentService) findCourseAndModule(courseUuid, moduleUuid uuid.UUID) (*course.Course, *course.Module, error) {
	var c course.Course
	if err := s.db.Where("uuid = ?", courseUuid).First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrCourseNotFound
		}
		return nil, nil, err
	}

	var m course.Module
	if err := s.db.Where("uuid = ? AND course_id = ?", moduleUuid, c.ID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrModuleNotFound
		}
		return nil, nil, err
	}
	m.Course = &c

	return &c, &m, nil
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
		Preload("File").
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

func (s *ContentService) findFile(moduleID uint, contentUuid uuid.UUID) (*File, error) {
	var cnt Content
	err := s.db.
		Where("uuid = ? AND module_id = ? AND type = ?", contentUuid, moduleID, TypeFile).
		Preload("File").
		Preload("Module.Course").
		First(&cnt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContentNotFound
		}
		return nil, err
	}
	if cnt.File == nil {
		return nil, ErrContentNotFound
	}
	cnt.File.Content = &cnt
	return cnt.File, nil
}

func (s *ContentService) UploadFile(
	courseUuid, moduleUuid uuid.UUID,
	title, fileName string,
	fileSize int64,
	contentType string,
	fileReader io.Reader,
	description string,
	sourceUrl string,
) (*File, error) {
	_, module, err := s.findCourseAndModule(courseUuid, moduleUuid)
	if err != nil {
		return nil, err
	}

	if fileName != "" {
		cleanName := filepath.Base(filepath.Clean(fileName))
		cleanName = strings.Map(func(r rune) rune {
			if r == '?' || r == '#' || r == '/' || r == '\\' || r == 0 {
				return '_'
			}
			return r
		}, cleanName)
		if len(cleanName) > 200 {
			cleanName = cleanName[:200]
		}
		fileName = cleanName
	}

	if title == "" {
		if fileName != "" {
			title = fileName
		} else {
			title = "Untitled Material"
		}
	}

	contentUuid := uuid.New()
	var storageKey string

	if fileReader != nil && fileName != "" {
		storageKey = fmt.Sprintf("content/%s/%s", contentUuid.String(), fileName)
		if err := s.storage.Upload(storageKey, fileReader, contentType); err != nil {
			s.logger.Errorf("Failed to upload file to storage: %v", err)
			return nil, err
		}
	}

	c := Content{
		Uuid:     contentUuid,
		ModuleID: module.ID,
		Title:    title,
		Type:     TypeFile,
	}

	f := File{
		FileName:    fileName,
		FileSize:    fileSize,
		ContentType: contentType,
		StorageKey:  storageKey,
		SourceURL:   sourceUrl,
		RichText:    description,
	}

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		var maxPos int
		row := tx.Model(&Content{}).Where("module_id = ?", module.ID).Select("COALESCE(MAX(position), 0)").Row()
		if err := row.Scan(&maxPos); err != nil {
			return err
		}
		c.Position = maxPos + 1

		if err := tx.Create(&c).Error; err != nil {
			return err
		}
		f.ContentID = c.ID
		if err := tx.Create(&f).Error; err != nil {
			return err
		}
		return nil
	})

	if txErr != nil {
		if storageKey != "" {
			_ = s.storage.Delete(storageKey)
		}
		return nil, txErr
	}

	c.Module = module
	f.Content = &c
	return &f, nil
}

func (s *ContentService) checkPrerequisiteCompleted(depID uint, userUuid uuid.UUID) error {
	var u user.User
	if err := s.db.Where("uuid = ?", userUuid).First(&u).Error; err != nil {
		return ErrModuleLocked
	}

	var totalContents int64
	if err := s.db.Model(&Content{}).Where("module_id = ?", depID).Count(&totalContents).Error; err != nil {
		return err
	}
	if totalContents == 0 {
		return nil
	}

	var completedContents int64
	err := s.db.Model(&StudentInteraction{}).
		Joins("JOIN contents ON contents.id = student_interactions.content_id").
		Where("contents.module_id = ? AND student_interactions.user_id = ? AND student_interactions.completed_at IS NOT NULL", depID, u.ID).
		Count(&completedContents).Error
	if err != nil {
		return err
	}

	if completedContents < totalContents {
		return ErrModuleLocked
	}
	return nil
}

func (s *ContentService) recordInteraction(userUuid uuid.UUID, contentID uint) {
	var u user.User
	if err := s.db.Where("uuid = ?", userUuid).First(&u).Error; err != nil {
		return
	}
	now := time.Now()
	var inter StudentInteraction
	err := s.db.Where("user_id = ? AND content_id = ?", u.ID, contentID).First(&inter).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		inter = StudentInteraction{
			UserID:      u.ID,
			UserEmail:   u.Email,
			ContentID:   contentID,
			CompletedAt: &now,
		}
		_ = s.db.Create(&inter).Error
	} else if err == nil && inter.CompletedAt == nil {
		inter.CompletedAt = &now
		_ = s.db.Save(&inter).Error
	}
}

func (s *ContentService) GetFile(courseUuid, moduleUuid, contentUuid, userUuid uuid.UUID, isPrivileged bool) (*File, error) {
	_, module, err := s.findCourseAndModule(courseUuid, moduleUuid)
	if err != nil {
		return nil, err
	}

	if !isPrivileged && module.DependsOnID != nil {
		if err := s.checkPrerequisiteCompleted(*module.DependsOnID, userUuid); err != nil {
			return nil, err
		}
	}

	f, err := s.findFile(module.ID, contentUuid)
	if err != nil {
		return nil, err
	}

	if !isPrivileged && f.Content != nil {
		now := time.Now()
		if (f.Content.VisibleFrom != nil && now.Before(*f.Content.VisibleFrom)) ||
			(f.Content.VisibleUntil != nil && now.After(*f.Content.VisibleUntil)) {
			return nil, ErrContentNotAvailable
		}
	}

	if !isPrivileged && userUuid != uuid.Nil && f.Content != nil {
		s.recordInteraction(userUuid, f.Content.ID)
	}

	return f, nil
}

func (s *ContentService) ListFiles(courseUuid, moduleUuid, userUuid uuid.UUID, isPrivileged bool) ([]File, error) {
	_, module, err := s.findCourseAndModule(courseUuid, moduleUuid)
	if err != nil {
		return nil, err
	}

	if !isPrivileged && module.DependsOnID != nil {
		if err := s.checkPrerequisiteCompleted(*module.DependsOnID, userUuid); err != nil {
			return nil, err
		}
	}

	query := s.db.Where("module_id = ? AND type = ?", module.ID, TypeFile)
	if !isPrivileged {
		now := time.Now()
		query = query.Where("(visible_from IS NULL OR visible_from <= ?)", now).
			Where("(visible_until IS NULL OR visible_until >= ?)", now)
	}

	var contents []Content
	err = query.
		Preload("File").
		Preload("Module.Course").
		Order("position ASC, id ASC").
		Find(&contents).Error
	if err != nil {
		return nil, err
	}

	files := make([]File, 0, len(contents))
	for i := range contents {
		if contents[i].File != nil {
			contents[i].File.Content = &contents[i]
			files = append(files, *contents[i].File)
		}
	}
	return files, nil
}

func (s *ContentService) GetFileStream(courseUuid, moduleUuid, contentUuid, userUuid uuid.UUID, isPrivileged bool) (io.ReadCloser, *File, error) {
	f, err := s.GetFile(courseUuid, moduleUuid, contentUuid, userUuid, isPrivileged)
	if err != nil {
		return nil, nil, err
	}

	if f.StorageKey == "" {
		return nil, nil, storage.ErrObjectNotFound
	}

	reader, _, err := s.storage.Get(f.StorageKey)
	if err != nil {
		return nil, nil, err
	}
	return reader, f, nil
}

func (s *ContentService) UpdateFile(courseUuid, moduleUuid, contentUuid uuid.UUID, title, description string) (*File, error) {
	f, err := s.GetFile(courseUuid, moduleUuid, contentUuid, uuid.Nil, true)
	if err != nil {
		return nil, err
	}

	if title != "" {
		f.Content.Title = title
	}
	f.RichText = description

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(f.Content).Error; err != nil {
			return err
		}
		return tx.Save(f).Error
	})
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (s *ContentService) DeleteFile(courseUuid, moduleUuid, contentUuid uuid.UUID) error {
	f, err := s.GetFile(courseUuid, moduleUuid, contentUuid, uuid.Nil, true)
	if err != nil {
		return err
	}

	if f.StorageKey != "" {
		if err := s.storage.Delete(f.StorageKey); err != nil {
			s.logger.Warnf("Failed to delete storage key %s: %v", f.StorageKey, err)
		}
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Delete(&File{}, "content_id = ?", f.ContentID).Error; err != nil {
			return err
		}
		return tx.Unscoped().Delete(&Content{}, "id = ?", f.Content.ID).Error
	})
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
