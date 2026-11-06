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
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/notification"
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
		&course.Enrollment{},
		&course.Module{},
		&content.Content{},
		&content.File{},
		&content.StudentInteraction{},
		&notification.Notification{},
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
	notifSvc := notification.NewNotificationService(db, notification.NewHub(), logger)
	svc := content.NewContentService(db, memStorage, notifSvc, logger)

	return c, m, svc, memStorage, db
}

func TestContentService_UploadFile_WithFile(t *testing.T) {
	c, m, svc, memStorage, _ := setupContentTest(t)

	fileContent := []byte("%PDF-1.4 test pdf content")
	reader := bytes.NewReader(fileContent)

	sp, err := svc.UploadFile(
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
	sp2, err := svc.UploadFile(
		c.Uuid, m.Uuid,
		"", "reading.md",
		10, "text/markdown",
		strings.NewReader("# Reading"), "Course reading notes", "",
	)
	require.NoError(t, err)
	assert.Equal(t, "reading.md", sp2.Content.Title) // default title to fileName
	assert.Equal(t, 2, sp2.Content.Position)
}

func TestContentService_UploadFile_DefaultUntitled(t *testing.T) {
	c, m, svc, _, _ := setupContentTest(t)

	sp, err := svc.UploadFile(
		c.Uuid, m.Uuid,
		"", "", 0, "",
		nil, "No title and no file", "http://example.com/link",
	)
	require.NoError(t, err)
	assert.Equal(t, "Untitled Material", sp.Content.Title)
}

func TestContentService_UpdateFile_EmptyTitle(t *testing.T) {
	c, m, svc, _, _ := setupContentTest(t)

	sp, err := svc.UploadFile(c.Uuid, m.Uuid, "Initial Title", "file.txt", 4, "text/plain", strings.NewReader("test"), "Initial Desc", "")
	require.NoError(t, err)

	updated, err := svc.UpdateFile(c.Uuid, m.Uuid, sp.Content.Uuid, "", "Only Desc Updated")
	require.NoError(t, err)
	assert.Equal(t, "Initial Title", updated.Content.Title)
	assert.Equal(t, "Only Desc Updated", updated.RichText)
}

func TestContentService_UploadFile_Failures(t *testing.T) {
	c, m, svc, memStorage, _ := setupContentTest(t)

	// Invalid course
	_, err := svc.UploadFile(
		uuid.New(), m.Uuid,
		"Title", "file.pdf", 10, "application/pdf",
		strings.NewReader("data"), "", "",
	)
	assert.ErrorIs(t, err, content.ErrCourseNotFound)

	// Invalid module
	_, err = svc.UploadFile(
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
	failingDb := testutil.NewSqliteTestDb(t, &user.User{}, &course.Course{}, &course.Enrollment{}, &course.Module{}, &content.Content{}, &content.File{}, &content.StudentInteraction{}, &notification.Notification{})
	require.NoError(t, failingDb.Create(c).Error)
	require.NoError(t, failingDb.Create(m).Error)
	failingSvc := content.NewContentService(failingDb, failingStorage, notification.NewNotificationService(failingDb, notification.NewHub(), zap.NewNop().Sugar()), zap.NewNop().Sugar())

	_, err = failingSvc.UploadFile(
		c.Uuid, m.Uuid,
		"Title", "fail.pdf", 10, "application/pdf",
		strings.NewReader("fail"), "", "",
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "simulated upload failure")
}

func TestContentService_GetAndListFiles(t *testing.T) {
	c, m, svc, _, db := setupContentTest(t)

	// Empty list
	pages, err := svc.ListFiles(c.Uuid, m.Uuid, uuid.Nil, true)
	require.NoError(t, err)
	assert.Empty(t, pages)

	// Upload two pages
	sp1, err := svc.UploadFile(c.Uuid, m.Uuid, "Page 1", "file1.txt", 5, "text/plain", strings.NewReader("hello"), "desc 1", "")
	require.NoError(t, err)

	sp2, err := svc.UploadFile(c.Uuid, m.Uuid, "Page 2", "file2.txt", 5, "text/plain", strings.NewReader("world"), "desc 2", "")
	require.NoError(t, err)

	// List
	list, err := svc.ListFiles(c.Uuid, m.Uuid, uuid.Nil, true)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, sp1.Content.Uuid, list[0].Content.Uuid)
	assert.Equal(t, sp2.Content.Uuid, list[1].Content.Uuid)

	// Get single
	fetched, err := svc.GetFile(c.Uuid, m.Uuid, sp1.Content.Uuid, uuid.Nil, true)
	require.NoError(t, err)
	assert.Equal(t, "Page 1", fetched.Content.Title)
	assert.Equal(t, "desc 1", fetched.RichText)

	// Not found checks
	_, err = svc.GetFile(c.Uuid, m.Uuid, uuid.New(), uuid.Nil, true)
	assert.ErrorIs(t, err, content.ErrContentNotFound)

	// Content exists but File extension row missing
	orphanCnt := &content.Content{
		Uuid:     uuid.New(),
		ModuleID: m.ID,
		Title:    "Orphan Content",
		Type:     content.TypeFile,
	}
	require.NoError(t, db.Create(orphanCnt).Error)
	_, err = svc.GetFile(c.Uuid, m.Uuid, orphanCnt.Uuid, uuid.Nil, true)
	assert.ErrorIs(t, err, content.ErrContentNotFound)

	_, err = svc.GetFile(uuid.New(), m.Uuid, sp1.Content.Uuid, uuid.Nil, true)
	assert.ErrorIs(t, err, content.ErrCourseNotFound)

	_, err = svc.GetFile(c.Uuid, uuid.New(), sp1.Content.Uuid, uuid.Nil, true)
	assert.ErrorIs(t, err, content.ErrModuleNotFound)

	_, err = svc.ListFiles(uuid.New(), m.Uuid, uuid.Nil, true)
	assert.ErrorIs(t, err, content.ErrCourseNotFound)

	_, err = svc.ListFiles(c.Uuid, uuid.New(), uuid.Nil, true)
	assert.ErrorIs(t, err, content.ErrModuleNotFound)
}

func TestContentService_GetFileStream(t *testing.T) {
	c, m, svc, memStorage, _ := setupContentTest(t)

	contentStr := "test stream content"
	sp, err := svc.UploadFile(c.Uuid, m.Uuid, "Stream Test", "stream.txt", int64(len(contentStr)), "text/plain", strings.NewReader(contentStr), "", "")
	require.NoError(t, err)

	rc, fetchedSp, err := svc.GetFileStream(c.Uuid, m.Uuid, sp.Content.Uuid, uuid.Nil, true)
	require.NoError(t, err)
	defer rc.Close()

	assert.Equal(t, sp.Content.Uuid, fetchedSp.Content.Uuid)
	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, contentStr, string(data))

	// Page without storage key
	spNoFile, err := svc.UploadFile(c.Uuid, m.Uuid, "No File", "", 0, "", nil, "", "http://example.com")
	require.NoError(t, err)
	_, _, err = svc.GetFileStream(c.Uuid, m.Uuid, spNoFile.Content.Uuid, uuid.Nil, true)
	assert.ErrorIs(t, err, storage.ErrObjectNotFound)

	// Storage error on Get
	failingStorage := &failingStorageService{
		IStorageService: memStorage,
		failGet:         true,
	}
	db := testutil.NewSqliteTestDb(t, &user.User{}, &course.Course{}, &course.Enrollment{}, &course.Module{}, &content.Content{}, &content.File{}, &content.StudentInteraction{}, &notification.Notification{})
	require.NoError(t, db.Create(c).Error)
	require.NoError(t, db.Create(m).Error)
	failingSvc := content.NewContentService(db, failingStorage, notification.NewNotificationService(db, notification.NewHub(), zap.NewNop().Sugar()), zap.NewNop().Sugar())
	spFail, err := failingSvc.UploadFile(c.Uuid, m.Uuid, "Will Fail", "fail.txt", 4, "text/plain", strings.NewReader("test"), "", "")
	require.NoError(t, err)

	_, _, err = failingSvc.GetFileStream(c.Uuid, m.Uuid, spFail.Content.Uuid, uuid.Nil, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "simulated get failure")
}

func TestContentService_UpdateFile(t *testing.T) {
	c, m, svc, _, _ := setupContentTest(t)

	sp, err := svc.UploadFile(c.Uuid, m.Uuid, "Initial Title", "file.txt", 4, "text/plain", strings.NewReader("test"), "Initial Desc", "")
	require.NoError(t, err)

	updated, err := svc.UpdateFile(c.Uuid, m.Uuid, sp.Content.Uuid, "Updated Title", "Updated Desc")
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", updated.Content.Title)
	assert.Equal(t, "Updated Desc", updated.RichText)

	// Not found
	_, err = svc.UpdateFile(c.Uuid, m.Uuid, uuid.New(), "New Title", "New Desc")
	assert.ErrorIs(t, err, content.ErrContentNotFound)
}

func TestContentService_DeleteFile(t *testing.T) {
	c, m, svc, memStorage, _ := setupContentTest(t)

	sp, err := svc.UploadFile(c.Uuid, m.Uuid, "To Delete", "delete.txt", 4, "text/plain", strings.NewReader("test"), "", "")
	require.NoError(t, err)

	// Verify in storage
	_, _, err = memStorage.Get(sp.StorageKey)
	require.NoError(t, err)

	// Delete
	err = svc.DeleteFile(c.Uuid, m.Uuid, sp.Content.Uuid)
	require.NoError(t, err)

	// Verify deleted from storage
	_, _, err = memStorage.Get(sp.StorageKey)
	assert.ErrorIs(t, err, storage.ErrObjectNotFound)

	// Verify deleted from DB
	_, err = svc.GetFile(c.Uuid, m.Uuid, sp.Content.Uuid, uuid.Nil, true)
	assert.ErrorIs(t, err, content.ErrContentNotFound)

	// Delete non-existent
	err = svc.DeleteFile(c.Uuid, m.Uuid, uuid.New())
	assert.ErrorIs(t, err, content.ErrContentNotFound)

	// Delete with failing storage delete (should still delete DB record and succeed)
	failingStorage := &failingStorageService{
		IStorageService: memStorage,
		failDelete:      true,
	}
	db := testutil.NewSqliteTestDb(t, &user.User{}, &course.Course{}, &course.Enrollment{}, &course.Module{}, &content.Content{}, &content.File{}, &content.StudentInteraction{}, &notification.Notification{})
	require.NoError(t, db.Create(c).Error)
	require.NoError(t, db.Create(m).Error)
	failingSvc := content.NewContentService(db, failingStorage, notification.NewNotificationService(db, notification.NewHub(), zap.NewNop().Sugar()), zap.NewNop().Sugar())

	sp2, err := failingSvc.UploadFile(c.Uuid, m.Uuid, "Delete Warn", "del.txt", 3, "text/plain", strings.NewReader("abc"), "", "")
	require.NoError(t, err)

	err = failingSvc.DeleteFile(c.Uuid, m.Uuid, sp2.Content.Uuid)
	assert.NoError(t, err)

	_, err = failingSvc.GetFile(c.Uuid, m.Uuid, sp2.Content.Uuid, uuid.Nil, true)
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
	page1, err := svc.UploadFile(c.Uuid, m1.Uuid, "Page 1", "m1.pdf", 10, "application/pdf", strings.NewReader("data"), "", "")
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
	page2, err := svc.UploadFile(c.Uuid, m2.Uuid, "Page 2", "m2.pdf", 10, "application/pdf", strings.NewReader("data2"), "", "")
	require.NoError(t, err)

	// Student attempts to list or get Module 2 without completing Module 1 -> ErrModuleLocked
	_, err = svc.ListFiles(c.Uuid, m2.Uuid, student.Uuid, false)
	assert.ErrorIs(t, err, content.ErrModuleLocked)

	_, err = svc.GetFile(c.Uuid, m2.Uuid, page2.Content.Uuid, student.Uuid, false)
	assert.ErrorIs(t, err, content.ErrModuleLocked)

	// Privileged teacher can still access Module 2
	pages, err := svc.ListFiles(c.Uuid, m2.Uuid, uuid.Nil, true)
	require.NoError(t, err)
	assert.Len(t, pages, 1)

	// Student views Module 1 content (records interaction)
	_, err = svc.GetFile(c.Uuid, m1.Uuid, page1.Content.Uuid, student.Uuid, false)
	require.NoError(t, err)

	// Viewing a second time updates existing interaction
	_, err = svc.GetFile(c.Uuid, m1.Uuid, page1.Content.Uuid, student.Uuid, false)
	require.NoError(t, err)

	// Now Module 2 is unlocked for student!
	pagesStudent, err := svc.ListFiles(c.Uuid, m2.Uuid, student.Uuid, false)
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
	pageHidden, err := svc.UploadFile(c.Uuid, m1.Uuid, "Hidden Page", "h.txt", 4, "text/plain", strings.NewReader("hide"), "", "")
	require.NoError(t, err)
	require.NoError(t, db.Model(&content.Content{}).Where("id = ?", pageHidden.ContentID).Update("visible_from", &future).Error)

	// Content with past VisibleUntil
	pageExpired, err := svc.UploadFile(c.Uuid, m1.Uuid, "Expired Page", "e.txt", 4, "text/plain", strings.NewReader("exp"), "", "")
	require.NoError(t, err)
	require.NoError(t, db.Model(&content.Content{}).Where("id = ?", pageExpired.ContentID).Update("visible_until", &past).Error)

	// Student cannot get hidden or expired content or stream it
	_, err = svc.GetFile(c.Uuid, m1.Uuid, pageHidden.Content.Uuid, student.Uuid, false)
	assert.ErrorIs(t, err, content.ErrContentNotAvailable)

	_, _, err = svc.GetFileStream(c.Uuid, m1.Uuid, pageHidden.Content.Uuid, student.Uuid, false)
	assert.ErrorIs(t, err, content.ErrContentNotAvailable)

	_, err = svc.GetFile(c.Uuid, m1.Uuid, pageExpired.Content.Uuid, student.Uuid, false)
	assert.ErrorIs(t, err, content.ErrContentNotAvailable)

	// Privileged teacher can still get hidden content
	gotHidden, err := svc.GetFile(c.Uuid, m1.Uuid, pageHidden.Content.Uuid, uuid.Nil, true)
	require.NoError(t, err)
	assert.NotNil(t, gotHidden)
}

func TestContentService_CreateAssignmentAndGet(t *testing.T) {
	db := testutil.NewSqliteTestDb(t,
		&user.User{},
		&course.Course{},
		&course.Enrollment{},
		&course.Module{},
		&content.Content{},
		&content.File{},
		&content.Assignment{},
		&content.Quiz{},
		&content.Question{},
		&content.CodingExercise{},
		&notification.Notification{},
	)

	svc := content.NewContentService(db, storage.NewMemoryStorageService(), notification.NewNotificationService(db, notification.NewHub(), zap.NewNop().Sugar()), zap.NewNop().Sugar())

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

	t.Run("CreateAssignment fails when assignment creation fails inside transaction", func(t *testing.T) {
		cnt := &content.Content{
			Title:    "Exercise Duplicate",
			Position: 2,
		}
		asgn := &content.Assignment{
			Type: content.AssignmentHandIn,
		}
		created, err := svc.CreateAssignment(m.Uuid, cnt, asgn)
		assert.NoError(t, err)

		// Drop assignments table to force tx.Create(asgn) failure
		assert.NoError(t, db.Migrator().DropTable(&content.Assignment{}))

		cnt2 := &content.Content{
			Title:    "Exercise Broken",
			Position: 3,
		}
		_, err = svc.CreateAssignment(m.Uuid, cnt2, &content.Assignment{Type: content.AssignmentHandIn})
		assert.Error(t, err)

		// Recreate table and test closed DB error paths
		assert.NoError(t, db.AutoMigrate(&content.Assignment{}))

		sqlDB, err := db.DB()
		assert.NoError(t, err)
		_ = sqlDB.Close()

		_, err = svc.CreateAssignment(m.Uuid, &content.Content{Title: "Closed"}, &content.Assignment{})
		assert.Error(t, err)

		_, err = svc.GetContent(created.Uuid)
		assert.Error(t, err)

		_, err = svc.GetAssignmentByModule(m.Uuid)
		assert.Error(t, err)
	})
}

func TestContentService_Notifications(t *testing.T) {
	db := testutil.NewSqliteTestDb(t,
		&user.User{},
		&course.Course{},
		&course.Enrollment{},
		&course.Module{},
		&content.Content{},
		&content.File{},
		&content.Assignment{},
		&notification.Notification{},
	)
	memStorage := storage.NewMemoryStorageService()
	logger := zap.NewNop().Sugar()
	notifSvc := notification.NewNotificationService(db, notification.NewHub(), logger)
	svc := content.NewContentService(db, memStorage, notifSvc, logger)

	c := &course.Course{Title: "Cloud Computing", Term: course.TermAutumn, Year: 2026}
	require.NoError(t, db.Create(c).Error)

	m := &course.Module{CourseID: c.ID, Title: "Module 1", Position: 1}
	require.NoError(t, db.Create(m).Error)

	student := &user.User{
		Uuid:     uuid.New(),
		Username: "stud",
		Email:    "stud@example.com",
	}
	teacher := &user.User{
		Uuid:     uuid.New(),
		Username: "prof",
		Email:    "prof@example.com",
	}
	require.NoError(t, db.Create(student).Error)
	require.NoError(t, db.Create(teacher).Error)

	require.NoError(t, db.Create(&course.Enrollment{
		UserID:   student.ID,
		CourseID: c.ID,
		Role:     course.RoleStudent,
	}).Error)
	require.NoError(t, db.Create(&course.Enrollment{
		UserID:   teacher.ID,
		CourseID: c.ID,
		Role:     course.RoleTeacher,
	}).Error)

	t.Run("UploadFile notifies enrolled student", func(t *testing.T) {
		f, err := svc.UploadFile(c.Uuid, m.Uuid, "Lecture 1 Slides", "l1.pdf", 100, "application/pdf", strings.NewReader("pdf-bytes"), "Intro", "")
		require.NoError(t, err)
		require.NotNil(t, f)

		var notifs []notification.Notification
		require.NoError(t, db.Where("user_id = ?", student.ID).Find(&notifs).Error)
		require.NotEmpty(t, notifs)
		assert.Equal(t, "New Course Material: Lecture 1 Slides", notifs[len(notifs)-1].Title)
		assert.Contains(t, notifs[len(notifs)-1].Body, "Lecture 1 Slides")
		assert.Contains(t, notifs[len(notifs)-1].URL, "/modules/"+m.Uuid.String()+"/files/")

		var teacherNotifs []notification.Notification
		require.NoError(t, db.Where("user_id = ?", teacher.ID).Find(&teacherNotifs).Error)
		assert.Empty(t, teacherNotifs)
	})

	t.Run("CreateAssignment notifies enrolled student", func(t *testing.T) {
		cnt := &content.Content{Title: "Homework 1", Position: 2}
		asgn := &content.Assignment{Type: content.AssignmentHandIn}
		created, err := svc.CreateAssignment(m.Uuid, cnt, asgn)
		require.NoError(t, err)
		require.NotNil(t, created)

		var notifs []notification.Notification
		require.NoError(t, db.Where("user_id = ?", student.ID).Find(&notifs).Error)
		assert.Equal(t, "New Assignment: Homework 1", notifs[len(notifs)-1].Title)
		assert.Contains(t, notifs[len(notifs)-1].Body, "Homework 1")
		assert.Equal(t, "/courses/"+c.Uuid.String(), notifs[len(notifs)-1].URL)
	})
}

func TestContentService_HandinAssignment(t *testing.T) {
	db := testutil.NewSqliteTestDb(t,
		&user.User{},
		&course.Course{},
		&course.Enrollment{},
		&course.Module{},
		&content.Content{},
		&content.Assignment{},
		&notification.Notification{},
	)

	memStorage := storage.NewMemoryStorageService()
	logger := zap.NewNop().Sugar()
	notifSvc := notification.NewNotificationService(db, notification.NewHub(), logger)
	svc := content.NewContentService(db, memStorage, notifSvc, logger)

	c := &course.Course{Title: "DAT250", Term: course.TermAutumn, Year: 2026}
	assert.NoError(t, db.Create(c).Error)

	m := &course.Module{CourseID: c.ID, Title: "Module 1", Position: 1}
	assert.NoError(t, db.Create(m).Error)

	deadline := time.Now().Add(48 * time.Hour)

	// 1. CreateHandin
	created, err := svc.CreateHandin(c.Uuid, m.Uuid, "Assignment 1", &deadline, true)
	assert.NoError(t, err)
	assert.NotNil(t, created)
	assert.Equal(t, "Assignment 1", created.Title)
	assert.Equal(t, 1, created.Position)
	assert.Equal(t, content.TypeAssignment, created.Type)
	assert.Equal(t, content.AssignmentHandIn, created.Assignment.Type)
	assert.True(t, created.Assignment.AcceptsLateSubmissions)

	// Create second handin to test auto-position
	created2, err := svc.CreateHandin(c.Uuid, m.Uuid, "Assignment 2", nil, false)
	assert.NoError(t, err)
	assert.Equal(t, 2, created2.Position)

	// 2. GetHandin
	fetched, err := svc.GetHandin(c.Uuid, m.Uuid, created.Uuid)
	assert.NoError(t, err)
	assert.Equal(t, created.Uuid, fetched.Uuid)
	assert.Equal(t, "Assignment 1", fetched.Title)

	// GetHandin not found
	_, err = svc.GetHandin(c.Uuid, m.Uuid, uuid.New())
	assert.ErrorIs(t, err, content.ErrContentNotFound)

	// 3. UpdateHandin
	newDeadline := time.Now().Add(72 * time.Hour)
	updated, err := svc.UpdateHandin(c.Uuid, m.Uuid, created.Uuid, "Assignment 1 Updated", &newDeadline, false)
	assert.NoError(t, err)
	assert.Equal(t, "Assignment 1 Updated", updated.Title)
	assert.False(t, updated.Assignment.AcceptsLateSubmissions)

	// 4. ListHandins
	list, err := svc.ListHandins(c.Uuid, m.Uuid)
	assert.NoError(t, err)
	assert.Len(t, list, 2)
	assert.Equal(t, "Assignment 1 Updated", list[0].Title)
	assert.Equal(t, "Assignment 2", list[1].Title)

	// 5. DeleteHandin
	err = svc.DeleteHandin(c.Uuid, m.Uuid, created2.Uuid)
	assert.NoError(t, err)

	listAfterDelete, err := svc.ListHandins(c.Uuid, m.Uuid)
	assert.NoError(t, err)
	assert.Len(t, listAfterDelete, 1)

	// Delete non-existent
	err = svc.DeleteHandin(c.Uuid, m.Uuid, uuid.New())
	assert.ErrorIs(t, err, content.ErrContentNotFound)
}
