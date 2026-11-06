package quiz_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/quiz"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// draftWithWindow creates a quiz with one question and the given window, but does not publish it.
func (f *fixture) draftWithWindow(t *testing.T, from, until *time.Time) uuid.UUID {
	dto, err := f.svc.CreateQuiz(f.course.Uuid, f.module.Uuid, "Quiz 1")
	require.NoError(t, err)
	cu := uuid.MustParse(dto.Uuid)
	_, err = f.svc.ReplaceQuestions(f.course.Uuid, cu, quiz.ReplaceQuestionsRequest{Questions: []quiz.QuestionInput{trueFalseQuestion}})
	require.NoError(t, err)
	_, err = f.svc.UpdateQuiz(f.course.Uuid, cu, quiz.UpdateQuizRequest{
		Title: "Quiz 1", VisibleFrom: from, VisibleUntil: until, PassPercent: 50, MaxAttempts: 1,
	})
	require.NoError(t, err)
	return cu
}

func timePtr(d time.Duration) *time.Time {
	t := time.Now().Add(d)
	return &t
}

func (f *fixture) addStudent(t *testing.T, name string) *user.User {
	u := &user.User{Uuid: uuid.New(), Username: name, Email: name + "@uib.no", PasswordHash: "x"}
	require.NoError(t, f.db.Create(u).Error)
	require.NoError(t, f.db.Create(&course.Enrollment{UserID: u.ID, CourseID: f.course.ID, Role: course.RoleStudent, EnrolledAt: time.Now()}).Error)
	return u
}

func TestNotify_PublishOpenQuizNotifiesEachStudentSeparately(t *testing.T) {
	f := setup(t)
	second := f.addStudent(t, "second")
	cu := f.draftWithWindow(t, timePtr(-24*time.Hour), timePtr(24*time.Hour))

	_, err := f.svc.Publish(f.course.Uuid, cu)
	require.NoError(t, err)

	require.Len(t, f.notes.sent[f.student.Uuid], 1)
	require.Len(t, f.notes.sent[second.Uuid], 1)
	assert.Empty(t, f.notes.sent[f.teacher.Uuid], "teachers are not notified")
	want := fmt.Sprintf("/courses/%s/modules/%s/quizzes/%s/take", f.course.Uuid, f.module.Uuid, cu)
	assert.Equal(t, want, f.notes.sent[f.student.Uuid][0].URL)
	assert.Contains(t, f.notes.sent[f.student.Uuid][0].Title, "Quiz 1")
}

func TestNotify_PublishWithoutWindowNotifiesNow(t *testing.T) {
	f := setup(t)
	cu := f.draftWithWindow(t, nil, nil)

	_, err := f.svc.Publish(f.course.Uuid, cu)
	require.NoError(t, err)

	assert.Len(t, f.notes.sent[f.student.Uuid], 1)
}

func TestNotify_FutureWindowWaitsUntilStart(t *testing.T) {
	f := setup(t)
	cu := f.draftWithWindow(t, timePtr(time.Hour), timePtr(48*time.Hour))

	_, err := f.svc.Publish(f.course.Uuid, cu)
	require.NoError(t, err)
	require.NoError(t, f.svc.NotifyDueQuizzes())
	assert.Empty(t, f.notes.sent, "window has not opened")

	// Move the start into the past: the next worker run announces it.
	_, err = f.svc.UpdateQuiz(f.course.Uuid, cu, quiz.UpdateQuizRequest{
		Title: "Quiz 1", VisibleFrom: timePtr(-time.Minute), VisibleUntil: timePtr(48 * time.Hour), PassPercent: 50, MaxAttempts: 1,
	})
	require.NoError(t, err)
	require.NoError(t, f.svc.NotifyDueQuizzes())
	assert.Len(t, f.notes.sent[f.student.Uuid], 1)
}

func TestNotify_ExpiredWindowSendsNothing(t *testing.T) {
	f := setup(t)
	cu := f.draftWithWindow(t, timePtr(-48*time.Hour), timePtr(-time.Hour))

	_, err := f.svc.Publish(f.course.Uuid, cu)
	require.NoError(t, err)

	assert.Empty(t, f.notes.sent)
}

func TestNotify_DraftIsNeverAnnounced(t *testing.T) {
	f := setup(t)
	f.draftWithWindow(t, timePtr(-time.Hour), nil)

	require.NoError(t, f.svc.NotifyDueQuizzes())

	assert.Empty(t, f.notes.sent)
}

func TestNotify_WorkerRunsDoNotRepeat(t *testing.T) {
	f := setup(t)
	cu := f.draftWithWindow(t, nil, nil)
	_, err := f.svc.Publish(f.course.Uuid, cu)
	require.NoError(t, err)

	require.NoError(t, f.svc.NotifyDueQuizzes())
	require.NoError(t, f.svc.NotifyDueQuizzes())

	assert.Len(t, f.notes.sent[f.student.Uuid], 1)
}

func TestNotify_RepublishNotifiesAgain(t *testing.T) {
	f := setup(t)
	cu := f.draftWithWindow(t, nil, nil)
	_, err := f.svc.Publish(f.course.Uuid, cu)
	require.NoError(t, err)

	_, err = f.svc.Publish(f.course.Uuid, cu)
	require.NoError(t, err)

	got := f.notes.sent[f.student.Uuid]
	require.Len(t, got, 2)
	assert.Contains(t, got[1].Title, "New quiz")
}

func TestNotify_EditOfActiveQuizSendsUpdateOnceForBothSaves(t *testing.T) {
	f := setup(t)
	cu := f.draftWithWindow(t, nil, nil)
	_, err := f.svc.Publish(f.course.Uuid, cu)
	require.NoError(t, err)

	// The editor saves settings and questions in two calls: students still get one message.
	_, err = f.svc.UpdateQuiz(f.course.Uuid, cu, quiz.UpdateQuizRequest{Title: "Quiz 1", PassPercent: 60, MaxAttempts: 2})
	require.NoError(t, err)
	_, err = f.svc.ReplaceQuestions(f.course.Uuid, cu, quiz.ReplaceQuestionsRequest{Questions: []quiz.QuestionInput{trueFalseQuestion, animalQuestion}})
	require.NoError(t, err)
	require.NoError(t, f.svc.NotifyDueQuizzes())
	require.NoError(t, f.svc.NotifyDueQuizzes())

	got := f.notes.sent[f.student.Uuid]
	require.Len(t, got, 2, "one for publish, one for the edit")
	assert.Contains(t, got[1].Title, "Quiz updated")
	assert.Empty(t, f.notes.sent[f.teacher.Uuid])
}

func TestNotify_EditOfDraftSendsNothing(t *testing.T) {
	f := setup(t)
	cu := f.draftWithWindow(t, nil, nil)

	_, err := f.svc.ReplaceQuestions(f.course.Uuid, cu, quiz.ReplaceQuestionsRequest{Questions: []quiz.QuestionInput{animalQuestion}})
	require.NoError(t, err)
	require.NoError(t, f.svc.NotifyDueQuizzes())

	assert.Empty(t, f.notes.sent)
}

func TestNotify_EditBeforeFirstAnnouncementSendsNewNotUpdate(t *testing.T) {
	f := setup(t)
	cu := f.draftWithWindow(t, timePtr(time.Hour), nil)
	_, err := f.svc.Publish(f.course.Uuid, cu)
	require.NoError(t, err)

	_, err = f.svc.ReplaceQuestions(f.course.Uuid, cu, quiz.ReplaceQuestionsRequest{Questions: []quiz.QuestionInput{animalQuestion}})
	require.NoError(t, err)
	_, err = f.svc.UpdateQuiz(f.course.Uuid, cu, quiz.UpdateQuizRequest{Title: "Quiz 1", VisibleFrom: timePtr(-time.Minute), PassPercent: 50, MaxAttempts: 1})
	require.NoError(t, err)
	require.NoError(t, f.svc.NotifyDueQuizzes())

	got := f.notes.sent[f.student.Uuid]
	require.Len(t, got, 1)
	assert.Contains(t, got[0].Title, "New quiz")
}

func TestNotify_EditOutsideWindowWaitsUntilOpen(t *testing.T) {
	f := setup(t)
	cu := f.draftWithWindow(t, nil, nil)
	_, err := f.svc.Publish(f.course.Uuid, cu)
	require.NoError(t, err)

	// Edit and move the window to the future: update notice waits for the start.
	_, err = f.svc.UpdateQuiz(f.course.Uuid, cu, quiz.UpdateQuizRequest{Title: "Quiz 1", VisibleFrom: timePtr(time.Hour), PassPercent: 50, MaxAttempts: 1})
	require.NoError(t, err)
	require.NoError(t, f.svc.NotifyDueQuizzes())
	assert.Len(t, f.notes.sent[f.student.Uuid], 1, "only the publish message so far")

	_, err = f.svc.UpdateQuiz(f.course.Uuid, cu, quiz.UpdateQuizRequest{Title: "Quiz 1", VisibleFrom: timePtr(-time.Minute), PassPercent: 50, MaxAttempts: 1})
	require.NoError(t, err)
	require.NoError(t, f.svc.NotifyDueQuizzes())
	got := f.notes.sent[f.student.Uuid]
	require.Len(t, got, 2)
	assert.Contains(t, got[1].Title, "Quiz updated")
}

func TestNotify_OneFailedDeliveryDoesNotBlockOthers(t *testing.T) {
	f := setup(t)
	second := f.addStudent(t, "second")
	f.notes.fail[f.student.Uuid] = true
	cu := f.draftWithWindow(t, nil, nil)

	_, err := f.svc.Publish(f.course.Uuid, cu)
	require.NoError(t, err)

	assert.Empty(t, f.notes.sent[f.student.Uuid])
	assert.Len(t, f.notes.sent[second.Uuid], 1)
}

func TestNotifyWorker_RunChecksOnStartAndStopsOnCancel(t *testing.T) {
	f := setup(t)
	f.draftWithWindow(t, timePtr(-time.Hour), nil)
	// Mark published directly so Publish does not announce it: only the worker can.
	require.NoError(t, f.db.Table("quizzes").Where("1 = 1").Update("published", true).Error)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		quiz.NewNotifyWorker(f.svc, zap.NewNop().Sugar()).Run(ctx)
		close(done)
	}()

	assert.Eventually(t, func() bool { return f.notes.count(f.student.Uuid) == 1 }, 2*time.Second, 10*time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not stop after cancel")
	}
}
