package content_test

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/content"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/testutil"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/storage"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type failingStorageService struct {
	storage.IStorageService
	failUpload bool
	failGet    bool
	failDelete bool
}

func (f *failingStorageService) Upload(key string, reader io.Reader, contentType string) error {
	if f.failUpload {
		return errors.New("simulated upload failure")
	}
	return f.IStorageService.Upload(key, reader, contentType)
}

func (f *failingStorageService) Get(key string) (io.ReadCloser, string, error) {
	if f.failGet {
		return nil, "", errors.New("simulated get failure")
	}
	return f.IStorageService.Get(key)
}

func (f *failingStorageService) Delete(key string) error {
	if f.failDelete {
		return errors.New("simulated delete failure")
	}
	return f.IStorageService.Delete(key)
}

func setupContentTest(t *testing.T) (*course.Course, *course.Module, content.IContentService, storage.IStorageService, *gorm.DB) {
	db := testutil.NewSqliteTestDb(t,
		&user.User{},
		&course.Course{},
		&course.Module{},
		&content.Content{},
		&content.StaticPage{},
		&content.StudentInteraction{},
	)

	c := &course.Course{
		Uuid:  uuid.New(),
		Title: "Distributed Systems",
		Term:  course.TermAutumn,
		Year:  2026,
	}
	require.NoError(t, db.Create(c).Error)

	m := &course.Module{
		Uuid:     uuid.New(),
		CourseID: c.ID,
		Title:    "Module 1: Introduction",
		Position: 1,
	}
	require.NoError(t, db.Create(m).Error)

	memStorage := storage.NewMemoryStorageService()
	logger := zap.NewNop().Sugar()
	svc := content.NewContentService(db, memStorage, logger)

	return c, m, svc, memStorage, db
}

func TestContentService_UploadStaticPage_WithFile(t *testing.T) {
	c, m, svc, memStorage, _ := setupContentTest(t)

	fileContent := []byte("%PDF-1.4 test pdf content")
	reader := bytes.NewReader(fileContent)

	sp, err := svc.UploadStaticPage(
		c.Uuid, m.Uuid,
		"Lecture 1 Slides", "lecture1.pdf",
		int64(len(fileContent)), "application/pdf",
		reader, "Lecture 1 slides overview", "",
	)
	require.NoError(t, err)
	assert.NotNil(t, sp)
	assert.Equal(t, "lecture1.pdf", sp.FileName)
	assert.Equal(t, int64(len(fileContent)), sp.FileSize)
	assert.Equal(t, "application/pdf", sp.ContentType)
	assert.Equal(t, "Lecture 1 slides overview", sp.RichText)
	assert.NotEmpty(t, sp.StorageKey)
	assert.Equal(t, "Lecture 1 Slides", sp.Content.Title)
	assert.Equal(t, 1, sp.Content.Position)

	// Verify file exists in storage
	rc, ct, err := memStorage.Get(sp.StorageKey)
	require.NoError(t, err)
	defer rc.Close()
	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, fileContent, data)
	assert.Equal(t, "application/pdf", ct)

	// Upload second file - verify position increments
	sp2, err := svc.UploadStaticPage(
		c.Uuid, m.Uuid,
		"", "reading.md",
		10, "text/markdown",
		strings.NewReader("# Reading"), "Course reading notes", "",
	)
	require.NoError(t, err)
	assert.Equal(t, "reading.md", sp2.Content.Title) // default title to fileName
	assert.Equal(t, 2, sp2.Content.Position)
}

func TestContentService_UploadStaticPage_DefaultUntitled(t *testing.T) {
	c, m, svc, _, _ := setupContentTest(t)

	sp, err := svc.UploadStaticPage(
		c.Uuid, m.Uuid,
		"", "", 0, "",
		nil, "No title and no file", "http://example.com/link",
	)
	require.NoError(t, err)
	assert.Equal(t, "Untitled Material", sp.Content.Title)
}

func TestContentService_UpdateStaticPage_EmptyTitle(t *testing.T) {
	c, m, svc, _, _ := setupContentTest(t)

	sp, err := svc.UploadStaticPage(c.Uuid, m.Uuid, "Initial Title", "file.txt", 4, "text/plain", strings.NewReader("test"), "Initial Desc", "")
	require.NoError(t, err)

	updated, err := svc.UpdateStaticPage(c.Uuid, m.Uuid, sp.Content.Uuid, "", "Only Desc Updated")
	require.NoError(t, err)
	assert.Equal(t, "Initial Title", updated.Content.Title)
	assert.Equal(t, "Only Desc Updated", updated.RichText)
}

func TestContentService_UploadStaticPage_Failures(t *testing.T) {
	c, m, svc, memStorage, _ := setupContentTest(t)

	// Invalid course
	_, err := svc.UploadStaticPage(
		uuid.New(), m.Uuid,
		"Title", "file.pdf", 10, "application/pdf",
		strings.NewReader("data"), "", "",
	)
	assert.ErrorIs(t, err, content.ErrCourseNotFound)

	// Invalid module
	_, err = svc.UploadStaticPage(
		c.Uuid, uuid.New(),
		"Title", "file.pdf", 10, "application/pdf",
		strings.NewReader("data"), "", "",
	)
	assert.ErrorIs(t, err, content.ErrModuleNotFound)

	// Storage failure
	failingStorage := &failingStorageService{
		IStorageService: memStorage,
		failUpload:      true,
	}
	failingDb := testutil.NewSqliteTestDb(t, &user.User{}, &course.Course{}, &course.Module{}, &content.Content{}, &content.StaticPage{}, &content.StudentInteraction{})
	require.NoError(t, failingDb.Create(c).Error)
	require.NoError(t, failingDb.Create(m).Error)
	failingSvc := content.NewContentService(failingDb, failingStorage, nil)

	_, err = failingSvc.UploadStaticPage(
		c.Uuid, m.Uuid,
		"Title", "fail.pdf", 10, "application/pdf",
		strings.NewReader("fail"), "", "",
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "simulated upload failure")
}

func TestContentService_GetAndListStaticPages(t *testing.T) {
	c, m, svc, _, db := setupContentTest(t)

	// Empty list
	pages, err := svc.ListStaticPages(c.Uuid, m.Uuid, uuid.Nil, true)
	require.NoError(t, err)
	assert.Empty(t, pages)

	// Upload two pages
	sp1, err := svc.UploadStaticPage(c.Uuid, m.Uuid, "Page 1", "file1.txt", 5, "text/plain", strings.NewReader("hello"), "desc 1", "")
	require.NoError(t, err)

	sp2, err := svc.UploadStaticPage(c.Uuid, m.Uuid, "Page 2", "file2.txt", 5, "text/plain", strings.NewReader("world"), "desc 2", "")
	require.NoError(t, err)

	// List
	list, err := svc.ListStaticPages(c.Uuid, m.Uuid, uuid.Nil, true)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, sp1.Content.Uuid, list[0].Content.Uuid)
	assert.Equal(t, sp2.Content.Uuid, list[1].Content.Uuid)

	// Get single
	fetched, err := svc.GetStaticPage(c.Uuid, m.Uuid, sp1.Content.Uuid, uuid.Nil, true)
	require.NoError(t, err)
	assert.Equal(t, "Page 1", fetched.Content.Title)
	assert.Equal(t, "desc 1", fetched.RichText)

	// Not found checks
	_, err = svc.GetStaticPage(c.Uuid, m.Uuid, uuid.New(), uuid.Nil, true)
	assert.ErrorIs(t, err, content.ErrContentNotFound)

	// Content exists but StaticPage extension row missing
	orphanCnt := &content.Content{
		Uuid:     uuid.New(),
		ModuleID: m.ID,
		Title:    "Orphan Content",
		Type:     content.TypeStaticPage,
	}
	require.NoError(t, db.Create(orphanCnt).Error)
	_, err = svc.GetStaticPage(c.Uuid, m.Uuid, orphanCnt.Uuid, uuid.Nil, true)
	assert.ErrorIs(t, err, content.ErrContentNotFound)

	_, err = svc.GetStaticPage(uuid.New(), m.Uuid, sp1.Content.Uuid, uuid.Nil, true)
	assert.ErrorIs(t, err, content.ErrCourseNotFound)

	_, err = svc.GetStaticPage(c.Uuid, uuid.New(), sp1.Content.Uuid, uuid.Nil, true)
	assert.ErrorIs(t, err, content.ErrModuleNotFound)

	_, err = svc.ListStaticPages(uuid.New(), m.Uuid, uuid.Nil, true)
	assert.ErrorIs(t, err, content.ErrCourseNotFound)

	_, err = svc.ListStaticPages(c.Uuid, uuid.New(), uuid.Nil, true)
	assert.ErrorIs(t, err, content.ErrModuleNotFound)
}

func TestContentService_GetFileStream(t *testing.T) {
	c, m, svc, memStorage, _ := setupContentTest(t)

	contentStr := "test stream content"
	sp, err := svc.UploadStaticPage(c.Uuid, m.Uuid, "Stream Test", "stream.txt", int64(len(contentStr)), "text/plain", strings.NewReader(contentStr), "", "")
	require.NoError(t, err)

	rc, fetchedSp, err := svc.GetFileStream(c.Uuid, m.Uuid, sp.Content.Uuid, uuid.Nil, true)
	require.NoError(t, err)
	defer rc.Close()

	assert.Equal(t, sp.Content.Uuid, fetchedSp.Content.Uuid)
	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, contentStr, string(data))

	// Page without storage key
	spNoFile, err := svc.UploadStaticPage(c.Uuid, m.Uuid, "No File", "", 0, "", nil, "", "http://example.com")
	require.NoError(t, err)
	_, _, err = svc.GetFileStream(c.Uuid, m.Uuid, spNoFile.Content.Uuid, uuid.Nil, true)
	assert.ErrorIs(t, err, storage.ErrObjectNotFound)

	// Storage error on Get
	failingStorage := &failingStorageService{
		IStorageService: memStorage,
		failGet:         true,
	}
	db := testutil.NewSqliteTestDb(t, &user.User{}, &course.Course{}, &course.Module{}, &content.Content{}, &content.StaticPage{}, &content.StudentInteraction{})
	require.NoError(t, db.Create(c).Error)
	require.NoError(t, db.Create(m).Error)
	failingSvc := content.NewContentService(db, failingStorage, nil)
	spFail, err := failingSvc.UploadStaticPage(c.Uuid, m.Uuid, "Will Fail", "fail.txt", 4, "text/plain", strings.NewReader("test"), "", "")
	require.NoError(t, err)

	_, _, err = failingSvc.GetFileStream(c.Uuid, m.Uuid, spFail.Content.Uuid, uuid.Nil, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "simulated get failure")
}

func TestContentService_UpdateStaticPage(t *testing.T) {
	c, m, svc, _, _ := setupContentTest(t)

	sp, err := svc.UploadStaticPage(c.Uuid, m.Uuid, "Initial Title", "file.txt", 4, "text/plain", strings.NewReader("test"), "Initial Desc", "")
	require.NoError(t, err)

	updated, err := svc.UpdateStaticPage(c.Uuid, m.Uuid, sp.Content.Uuid, "Updated Title", "Updated Desc")
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", updated.Content.Title)
	assert.Equal(t, "Updated Desc", updated.RichText)

	// Not found
	_, err = svc.UpdateStaticPage(c.Uuid, m.Uuid, uuid.New(), "New Title", "New Desc")
	assert.ErrorIs(t, err, content.ErrContentNotFound)
}

func TestContentService_DeleteStaticPage(t *testing.T) {
	c, m, svc, memStorage, _ := setupContentTest(t)

	sp, err := svc.UploadStaticPage(c.Uuid, m.Uuid, "To Delete", "delete.txt", 4, "text/plain", strings.NewReader("test"), "", "")
	require.NoError(t, err)

	// Verify in storage
	_, _, err = memStorage.Get(sp.StorageKey)
	require.NoError(t, err)

	// Delete
	err = svc.DeleteStaticPage(c.Uuid, m.Uuid, sp.Content.Uuid)
	require.NoError(t, err)

	// Verify deleted from storage
	_, _, err = memStorage.Get(sp.StorageKey)
	assert.ErrorIs(t, err, storage.ErrObjectNotFound)

	// Verify deleted from DB
	_, err = svc.GetStaticPage(c.Uuid, m.Uuid, sp.Content.Uuid, uuid.Nil, true)
	assert.ErrorIs(t, err, content.ErrContentNotFound)

	// Delete non-existent
	err = svc.DeleteStaticPage(c.Uuid, m.Uuid, uuid.New())
	assert.ErrorIs(t, err, content.ErrContentNotFound)

	// Delete with failing storage delete (should still delete DB record and succeed)
	failingStorage := &failingStorageService{
		IStorageService: memStorage,
		failDelete:      true,
	}
	db := testutil.NewSqliteTestDb(t, &user.User{}, &course.Course{}, &course.Module{}, &content.Content{}, &content.StaticPage{}, &content.StudentInteraction{})
	require.NoError(t, db.Create(c).Error)
	require.NoError(t, db.Create(m).Error)
	failingSvc := content.NewContentService(db, failingStorage, nil)

	sp2, err := failingSvc.UploadStaticPage(c.Uuid, m.Uuid, "Delete Warn", "del.txt", 3, "text/plain", strings.NewReader("abc"), "", "")
	require.NoError(t, err)

	err = failingSvc.DeleteStaticPage(c.Uuid, m.Uuid, sp2.Content.Uuid)
	assert.NoError(t, err)

	_, err = failingSvc.GetStaticPage(c.Uuid, m.Uuid, sp2.Content.Uuid, uuid.Nil, true)
	assert.ErrorIs(t, err, content.ErrContentNotFound)
}

func TestContentService_StudentPrerequisitesAndVisibility(t *testing.T) {
	c, m1, svc, _, db := setupContentTest(t)

	// Create student user
	student := &user.User{
		Uuid:     uuid.New(),
		Username: "alice",
		Email:    "alice@hvl.no",
	}
	require.NoError(t, db.Create(student).Error)

	// Module 1 content
	page1, err := svc.UploadStaticPage(c.Uuid, m1.Uuid, "Page 1", "m1.pdf", 10, "application/pdf", strings.NewReader("data"), "", "")
	require.NoError(t, err)

	// Module 2 depending on Module 1
	m2 := &course.Module{
		Uuid:        uuid.New(),
		CourseID:    c.ID,
		Title:       "Module 2: Advanced",
		Position:    2,
		DependsOnID: &m1.ID,
	}
	require.NoError(t, db.Create(m2).Error)

	// Module 2 content
	page2, err := svc.UploadStaticPage(c.Uuid, m2.Uuid, "Page 2", "m2.pdf", 10, "application/pdf", strings.NewReader("data2"), "", "")
	require.NoError(t, err)

	// Student attempts to list or get Module 2 without completing Module 1 -> ErrModuleLocked
	_, err = svc.ListStaticPages(c.Uuid, m2.Uuid, student.Uuid, false)
	assert.ErrorIs(t, err, content.ErrModuleLocked)

	_, err = svc.GetStaticPage(c.Uuid, m2.Uuid, page2.Content.Uuid, student.Uuid, false)
	assert.ErrorIs(t, err, content.ErrModuleLocked)

	// Privileged teacher can still access Module 2
	pages, err := svc.ListStaticPages(c.Uuid, m2.Uuid, uuid.Nil, true)
	require.NoError(t, err)
	assert.Len(t, pages, 1)

	// Student views Module 1 content (records interaction)
	_, err = svc.GetStaticPage(c.Uuid, m1.Uuid, page1.Content.Uuid, student.Uuid, false)
	require.NoError(t, err)

	// Viewing a second time updates existing interaction
	_, err = svc.GetStaticPage(c.Uuid, m1.Uuid, page1.Content.Uuid, student.Uuid, false)
	require.NoError(t, err)

	// Now Module 2 is unlocked for student!
	pagesStudent, err := svc.ListStaticPages(c.Uuid, m2.Uuid, student.Uuid, false)
	require.NoError(t, err)
	assert.Len(t, pagesStudent, 1)

	// Stream file from Module 2 succeeds
	rc, _, err := svc.GetFileStream(c.Uuid, m2.Uuid, page2.Content.Uuid, student.Uuid, false)
	require.NoError(t, err)
	_ = rc.Close()

	// Test Visibility Window
	future := time.Now().Add(24 * time.Hour)
	past := time.Now().Add(-24 * time.Hour)

	// Content with future VisibleFrom
	pageHidden, err := svc.UploadStaticPage(c.Uuid, m1.Uuid, "Hidden Page", "h.txt", 4, "text/plain", strings.NewReader("hide"), "", "")
	require.NoError(t, err)
	require.NoError(t, db.Model(&content.Content{}).Where("id = ?", pageHidden.ContentID).Update("visible_from", &future).Error)

	// Content with past VisibleUntil
	pageExpired, err := svc.UploadStaticPage(c.Uuid, m1.Uuid, "Expired Page", "e.txt", 4, "text/plain", strings.NewReader("exp"), "", "")
	require.NoError(t, err)
	require.NoError(t, db.Model(&content.Content{}).Where("id = ?", pageExpired.ContentID).Update("visible_until", &past).Error)

	// Student cannot get hidden or expired content or stream it
	_, err = svc.GetStaticPage(c.Uuid, m1.Uuid, pageHidden.Content.Uuid, student.Uuid, false)
	assert.ErrorIs(t, err, content.ErrContentNotAvailable)

	_, _, err = svc.GetFileStream(c.Uuid, m1.Uuid, pageHidden.Content.Uuid, student.Uuid, false)
	assert.ErrorIs(t, err, content.ErrContentNotAvailable)

	_, err = svc.GetStaticPage(c.Uuid, m1.Uuid, pageExpired.Content.Uuid, student.Uuid, false)
	assert.ErrorIs(t, err, content.ErrContentNotAvailable)

	// Privileged teacher can still get hidden content
	gotHidden, err := svc.GetStaticPage(c.Uuid, m1.Uuid, pageHidden.Content.Uuid, uuid.Nil, true)
	require.NoError(t, err)
	assert.NotNil(t, gotHidden)
}
