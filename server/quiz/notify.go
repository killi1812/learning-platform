package quiz

import (
	"context"
	"fmt"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/notification"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// notifyInterval is how often the worker looks for quizzes whose window has opened.
const notifyInterval = time.Minute

// Notifier delivers one notification to one user. notification.INotificationService satisfies it.
type Notifier interface {
	NotifySystem(recipientUuid uuid.UUID, content notification.NotificationContent) (*notification.Notification, error)
}

// dueQuiz is a published, not yet announced quiz whose window is open now.
type dueQuiz struct {
	ContentID  uint
	Title      string
	CourseID   uint
	CourseUuid uuid.UUID
	ModuleUuid uuid.UUID
	QuizUuid   uuid.UUID
	IsUpdate   bool
}

// rearmUpdateNotice queues a "quiz updated" announcement after a teacher edits a published quiz
// that students were already told about. A quiz that is still waiting for its first announcement
// is left alone, so students get one message with the final content. Several edits in a row
// (settings, then questions) collapse into one notification sent on the next worker run.
func rearmUpdateNotice(tx *gorm.DB, quizID uint) error {
	return tx.Table("quizzes").
		Where("content_id = ? AND published = ? AND notified_at IS NOT NULL", quizID, true).
		Updates(map[string]any{"notified_at": nil, "notify_update": true}).Error
}

// NotifyDueQuizzes tells every enrolled student about each published quiz that is open now and
// has not been announced yet. A quiz is announced once per publish: Publish clears NotifiedAt.
// Quizzes with a start date in the future are picked up on a later call, once the start passes.
func (s *QuizService) NotifyDueQuizzes() error {
	now := time.Now()

	var due []dueQuiz
	err := s.db.Table("quizzes").
		Select("quizzes.content_id, contents.title, modules.course_id, courses.uuid AS course_uuid, modules.uuid AS module_uuid, contents.uuid AS quiz_uuid, quizzes.notify_update AS is_update").
		Joins("JOIN contents ON contents.id = quizzes.content_id AND contents.deleted_at IS NULL").
		Joins("JOIN modules ON modules.id = contents.module_id").
		Joins("JOIN courses ON courses.id = modules.course_id").
		Where("quizzes.published = ? AND quizzes.notified_at IS NULL", true).
		Where("contents.visible_from IS NULL OR contents.visible_from <= ?", now).
		Where("contents.visible_until IS NULL OR contents.visible_until > ?", now).
		Scan(&due).Error
	if err != nil {
		return fmt.Errorf("find quizzes to announce: %w", err)
	}

	for _, q := range due {
		s.announce(q, now)
	}
	return nil
}

// announce claims the quiz, then sends one notification per student. The claim is an atomic
// update, so two callers (publish and the worker) never announce the same quiz twice.
func (s *QuizService) announce(q dueQuiz, now time.Time) {
	claim := s.db.Table("quizzes").
		Where("content_id = ? AND notified_at IS NULL", q.ContentID).
		Update("notified_at", now)
	if claim.Error != nil {
		s.logger.Errorf("Failed to claim quiz %s for notification: %v", q.QuizUuid, claim.Error)
		return
	}
	if claim.RowsAffected == 0 {
		return
	}

	studentUuids, err := s.studentUuids(q.CourseID)
	if err != nil {
		s.logger.Errorf("Failed to load students for quiz %s: %v", q.QuizUuid, err)
		return
	}

	content := notification.NotificationContent{
		Title: "New quiz: " + q.Title,
		Body:  "A quiz is open now in your course. Open it to start.",
		URL:   fmt.Sprintf("/courses/%s/modules/%s/quizzes/%s/take", q.CourseUuid, q.ModuleUuid, q.QuizUuid),
	}
	if q.IsUpdate {
		content.Title = "Quiz updated: " + q.Title
		content.Body = "Your teacher changed this quiz. Earlier attempts were reset, so take it again."
	}
	sent := 0
	for _, studentUuid := range studentUuids {
		if _, err := s.notifier.NotifySystem(studentUuid, content); err != nil {
			s.logger.Warnf("Failed to notify student %s about quiz %s: %v", studentUuid, q.QuizUuid, err)
			continue
		}
		sent++
	}
	s.logger.Infof("Notified %d of %d students about quiz %s", sent, len(studentUuids), q.QuizUuid)
}

// studentUuids returns the uuid of every user enrolled in the course as a student.
func (s *QuizService) studentUuids(courseID uint) ([]uuid.UUID, error) {
	var uuids []uuid.UUID
	err := s.db.Table("enrollments").
		Joins("JOIN users ON users.id = enrollments.user_id").
		Where("enrollments.course_id = ? AND enrollments.role = ? AND enrollments.deleted_at IS NULL", courseID, course.RoleStudent).
		Pluck("users.uuid", &uuids).Error
	return uuids, err
}

// NotifyWorker announces quizzes when their start date arrives.
type NotifyWorker struct {
	service IQuizService
	logger  *zap.SugaredLogger
}

func NewNotifyWorker(service IQuizService, logger *zap.SugaredLogger) app.Worker {
	return &NotifyWorker{service: service, logger: logger}
}

// Run checks once at start, then every notifyInterval, until ctx is canceled.
func (w *NotifyWorker) Run(ctx context.Context) {
	w.logger.Infof("Starting quiz notification worker, interval %s", notifyInterval)
	ticker := time.NewTicker(notifyInterval)
	defer ticker.Stop()

	for {
		if err := w.service.NotifyDueQuizzes(); err != nil {
			w.logger.Errorf("Quiz notification check failed: %v", err)
		}
		select {
		case <-ctx.Done():
			w.logger.Infof("Quiz notification worker stopped")
			return
		case <-ticker.C:
		}
	}
}
