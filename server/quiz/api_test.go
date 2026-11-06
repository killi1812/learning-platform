package quiz_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/quiz"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/storage"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	app.AccessKey = "test-quiz-secret-access-key-12345"
	app.RefreshKey = "test-quiz-secret-refresh-key-12345"
}

func (f *fixture) router() http.Handler {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	courseSvc := course.NewCourseService(f.db, storage.NewMemoryStorageService(), zap.NewNop().Sugar())
	quiz.NewQuizCtn(f.svc, courseSvc, zap.NewNop().Sugar()).RegisterEndpoints(r.Group("/api"))
	return r
}

func authFor(t *testing.T, u *user.User) string {
	token, _, err := authutils.GenerateTokens(u.Email, u.Username, authutils.ROLE_USER, u.Uuid)
	require.NoError(t, err)
	return "Bearer " + token
}

func call(r http.Handler, method, path, auth string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req, _ := http.NewRequest(method, path, &buf)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestApi_TeacherEndpointsRejectStudentsAndAnonymous(t *testing.T) {
	f := setup(t)
	q := f.publishedQuiz(t, trueFalseQuestion)
	r := f.router()
	base := "/api/course/" + f.course.Uuid.String() + "/quiz/" + q.Uuid

	student := authFor(t, f.student)
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, base},
		{http.MethodPut, base},
		{http.MethodDelete, base},
		{http.MethodPut, base + "/questions"},
		{http.MethodPost, base + "/publish"},
		{http.MethodGet, base + "/results"},
		{http.MethodPost, "/api/course/" + f.course.Uuid.String() + "/module/" + f.module.Uuid.String() + "/quiz"},
	} {
		assert.Equal(t, http.StatusForbidden, call(r, tc.method, tc.path, student, map[string]any{}).Code, "%s %s student", tc.method, tc.path)
		assert.Equal(t, http.StatusUnauthorized, call(r, tc.method, tc.path, "", nil).Code, "%s %s anonymous", tc.method, tc.path)
	}
}

func TestApi_TeacherCanCreateAndSeesAnswerKey(t *testing.T) {
	f := setup(t)
	r := f.router()
	teacher := authFor(t, f.teacher)

	w := call(r, http.MethodPost, "/api/course/"+f.course.Uuid.String()+"/module/"+f.module.Uuid.String()+"/quiz", teacher, map[string]string{"title": "New quiz"})
	require.Equal(t, http.StatusCreated, w.Code)
	var created quiz.EditQuizDto
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))

	w = call(r, http.MethodPut, "/api/course/"+f.course.Uuid.String()+"/quiz/"+created.Uuid+"/questions", teacher,
		quiz.ReplaceQuestionsRequest{Questions: []quiz.QuestionInput{animalQuestion}})
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"correct":[1,2]`)
}

func TestApi_StudentResponsesLeakNothing(t *testing.T) {
	f := setup(t)
	q := f.publishedQuiz(t, animalQuestion)
	r := f.router()
	student := authFor(t, f.student)
	base := "/api/course/" + f.course.Uuid.String() + "/quiz/" + q.Uuid

	w := call(r, http.MethodPost, base+"/attempt", student, nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "correct")
	assert.NotContains(t, w.Body.String(), "answerKey")

	var attempt quiz.AttemptDto
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &attempt))
	w = call(r, http.MethodPut, base+"/attempt/answers/"+attempt.Questions[0].Uuid, student, quiz.SaveAnswerRequest{Selected: []int{1, 3}})
	require.Equal(t, http.StatusNoContent, w.Code)

	w = call(r, http.MethodPost, base+"/attempt/submit", student, nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"percent":50,"passed":true,"attemptsUsed":1,"maxAttempts":1}`, w.Body.String())

	w = call(r, http.MethodPost, base+"/attempt", student, nil)
	assert.Equal(t, http.StatusConflict, w.Code, "attempt limit reached")
}

func TestApi_SubmitWithSkippedQuestionsGradesZero(t *testing.T) {
	f := setup(t)
	q := f.publishedQuiz(t, animalQuestion)
	r := f.router()
	student := authFor(t, f.student)
	base := "/api/course/" + f.course.Uuid.String() + "/quiz/" + q.Uuid

	require.Equal(t, http.StatusOK, call(r, http.MethodPost, base+"/attempt", student, nil).Code)
	w := call(r, http.MethodPost, base+"/attempt/submit", student, nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"percent":0,"passed":false,"attemptsUsed":1,"maxAttempts":1}`, w.Body.String())
}

func TestApi_AssignmentListForEnrolledOnly(t *testing.T) {
	f := setup(t)
	f.publishedQuiz(t, trueFalseQuestion)
	r := f.router()
	path := "/api/course/" + f.course.Uuid.String() + "/assignment"

	assert.Equal(t, http.StatusOK, call(r, http.MethodGet, path, authFor(t, f.student), nil).Code)

	outsider := &user.User{Uuid: uuid.New(), Username: "outsider", Email: "outsider@uib.no", PasswordHash: "x"}
	require.NoError(t, f.db.Create(outsider).Error)
	assert.Equal(t, http.StatusForbidden, call(r, http.MethodGet, path, authFor(t, outsider), nil).Code, "not enrolled")
}
