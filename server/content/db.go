package content

import (
	"errors"
	"fmt"
	"io"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/storage"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrContentNotFound = errors.New("content not found")
	ErrModuleNotFound  = errors.New("module not found")
	ErrCourseNotFound  = errors.New("course not found")
)

type IContentService interface {
	UploadStaticPage(courseUuid, moduleUuid uuid.UUID, title, fileName string, fileSize int64, contentType string, fileReader io.Reader, description string, sourceUrl string) (*StaticPage, error)
	GetStaticPage(courseUuid, moduleUuid, contentUuid uuid.UUID) (*StaticPage, error)
	ListStaticPages(courseUuid, moduleUuid uuid.UUID) ([]StaticPage, error)
	GetFileStream(courseUuid, moduleUuid, contentUuid uuid.UUID) (io.ReadCloser, *StaticPage, error)
	UpdateStaticPage(courseUuid, moduleUuid, contentUuid uuid.UUID, title, description string) (*StaticPage, error)
	DeleteStaticPage(courseUuid, moduleUuid, contentUuid uuid.UUID) error
}

type ContentService struct {
	db      *gorm.DB
	storage storage.IStorageService
	logger  *zap.SugaredLogger
}

func NewContentService(db *gorm.DB, storage storage.IStorageService, logger *zap.SugaredLogger) IContentService {
	if logger == nil {
		logger = zap.S()
	}
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

func (s *ContentService) findStaticPage(moduleID uint, contentUuid uuid.UUID) (*StaticPage, error) {
	var cnt Content
	err := s.db.
		Where("uuid = ? AND module_id = ? AND type = ?", contentUuid, moduleID, TypeStaticPage).
		Preload("StaticPage").
		Preload("Module.Course").
		First(&cnt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContentNotFound
		}
		return nil, err
	}
	if cnt.StaticPage == nil {
		return nil, ErrContentNotFound
	}
	cnt.StaticPage.Content = &cnt
	return cnt.StaticPage, nil
}

func (s *ContentService) UploadStaticPage(
	courseUuid, moduleUuid uuid.UUID,
	title, fileName string,
	fileSize int64,
	contentType string,
	fileReader io.Reader,
	description string,
	sourceUrl string,
) (*StaticPage, error) {
	_, module, err := s.findCourseAndModule(courseUuid, moduleUuid)
	if err != nil {
		return nil, err
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

	var maxPos int
	row := s.db.Model(&Content{}).Where("module_id = ?", module.ID).Select("COALESCE(MAX(position), 0)").Row()
	_ = row.Scan(&maxPos)

	c := Content{
		Uuid:     contentUuid,
		ModuleID: module.ID,
		Title:    title,
		Position: maxPos + 1,
		Type:     TypeStaticPage,
	}

	sp := StaticPage{
		FileName:    fileName,
		FileSize:    fileSize,
		ContentType: contentType,
		StorageKey:  storageKey,
		SourceURL:   sourceUrl,
		RichText:    description,
	}

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&c).Error; err != nil {
			return err
		}
		sp.ContentID = c.ID
		if err := tx.Create(&sp).Error; err != nil {
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
	sp.Content = &c
	return &sp, nil
}

func (s *ContentService) GetStaticPage(courseUuid, moduleUuid, contentUuid uuid.UUID) (*StaticPage, error) {
	_, module, err := s.findCourseAndModule(courseUuid, moduleUuid)
	if err != nil {
		return nil, err
	}

	return s.findStaticPage(module.ID, contentUuid)
}

func (s *ContentService) ListStaticPages(courseUuid, moduleUuid uuid.UUID) ([]StaticPage, error) {
	_, module, err := s.findCourseAndModule(courseUuid, moduleUuid)
	if err != nil {
		return nil, err
	}

	var contents []Content
	err = s.db.
		Where("module_id = ? AND type = ?", module.ID, TypeStaticPage).
		Preload("StaticPage").
		Preload("Module.Course").
		Order("position ASC, id ASC").
		Find(&contents).Error
	if err != nil {
		return nil, err
	}

	pages := make([]StaticPage, 0, len(contents))
	for i := range contents {
		if contents[i].StaticPage != nil {
			contents[i].StaticPage.Content = &contents[i]
			pages = append(pages, *contents[i].StaticPage)
		}
	}
	return pages, nil
}

func (s *ContentService) GetFileStream(courseUuid, moduleUuid, contentUuid uuid.UUID) (io.ReadCloser, *StaticPage, error) {
	sp, err := s.GetStaticPage(courseUuid, moduleUuid, contentUuid)
	if err != nil {
		return nil, nil, err
	}

	if sp.StorageKey == "" {
		return nil, nil, storage.ErrObjectNotFound
	}

	reader, _, err := s.storage.Get(sp.StorageKey)
	if err != nil {
		return nil, nil, err
	}
	return reader, sp, nil
}

func (s *ContentService) UpdateStaticPage(
	courseUuid, moduleUuid, contentUuid uuid.UUID,
	title, description string,
) (*StaticPage, error) {
	sp, err := s.GetStaticPage(courseUuid, moduleUuid, contentUuid)
	if err != nil {
		return nil, err
	}

	if title != "" {
		sp.Content.Title = title
	}
	sp.RichText = description

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(sp.Content).Error; err != nil {
			return err
		}
		return tx.Save(sp).Error
	})
	if err != nil {
		return nil, err
	}
	return sp, nil
}

func (s *ContentService) DeleteStaticPage(courseUuid, moduleUuid, contentUuid uuid.UUID) error {
	sp, err := s.GetStaticPage(courseUuid, moduleUuid, contentUuid)
	if err != nil {
		return err
	}

	if sp.StorageKey != "" {
		if err := s.storage.Delete(sp.StorageKey); err != nil {
			s.logger.Warnf("Failed to delete storage key %s: %v", sp.StorageKey, err)
		}
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Delete(&StaticPage{}, "content_id = ?", sp.ContentID).Error; err != nil {
			return err
		}
		return tx.Unscoped().Delete(&Content{}, "id = ?", sp.Content.ID).Error
	})
}
