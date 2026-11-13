package quiz

import (
	"errors"
	"net/http"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/ginutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type QuizCtn struct {
	quizService   IQuizService
	courseService course.ICourseService
	logger        *zap.SugaredLogger
}

func NewQuizCtn(quizService IQuizService, courseService course.ICourseService, logger *zap.SugaredLogger) app.Api {
	return &QuizCtn{quizService: quizService, courseService: courseService, logger: logger}
}

func (ctn *QuizCtn) RegisterEndpoints(api *gin.RouterGroup) {
	g := api.Group("/course/:uuid")
	g.Use(authutils.Protect())

	g.GET("/assignment", ctn.listAssignments)
	g.POST("/module/:moduleUuid/quiz", ctn.create)

	q := g.Group("/quiz/:contentUuid")
	q.GET("", ctn.getForEdit)
	q.PUT("", ctn.update)
	q.DELETE("", ctn.delete)
	q.PUT("/questions", ctn.replaceQuestions)
	q.POST("/publish", ctn.publish)
	q.GET("/results", ctn.results)

	q.POST("/attempt", ctn.startAttempt)
	q.GET("/attempt", ctn.getAttempt)
	q.PUT("/attempt/answers/:questionUuid", ctn.saveAnswer)
	q.POST("/attempt/submit", ctn.submit)
}

func (ctn *QuizCtn) handleError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, ErrQuizNotFound),
		errors.Is(err, ErrModuleNotFound),
		errors.Is(err, ErrQuestionNotFound),
		errors.Is(err, ErrNoActiveAttempt),
		errors.Is(err, ErrUserNotFound),
		errors.Is(err, course.ErrCourseNotFound):
		ctn.logger.Errorf("Quiz not found: %v", err)
		ginutil.RespondWithError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrQuizClosed):
		ctn.logger.Errorf("Quiz forbidden: %v", err)
		ginutil.RespondWithError(c, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrNoAttemptsLeft):
		ctn.logger.Errorf("Quiz conflict: %v", err)
		ginutil.RespondWithError(c, http.StatusConflict, err.Error())
	case errors.Is(err, ErrInvalidQuestion),
		errors.Is(err, ErrInvalidAnswer),
		errors.Is(err, ErrInvalidQuiz),
		errors.Is(err, ErrNoQuestions):
		ctn.logger.Errorf("Quiz bad request: %v", err)
		ginutil.RespondWithError(c, http.StatusBadRequest, err.Error())
	default:
		ctn.logger.Errorf("Unexpected quiz error: %v", err)
		ginutil.RespondWithError(c, http.StatusInternalServerError, "Internal server error")
	}
	return true
}

// teacherParams parses course and quiz uuids and requires teacher or admin access.
func (ctn *QuizCtn) teacherParams(c *gin.Context) (courseUuid, contentUuid uuid.UUID, ok bool) {
	courseUuid, ok = ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}
	contentUuid, ok = ginutil.ParseUUIDParam(c, "contentUuid")
	if !ok {
		return
	}
	ok = authutils.CheckCourseAccess(c, courseUuid, ctn.courseService.IsTeacherOrAdmin, course.ErrCourseNotFound)
	return
}

// studentParams parses course and quiz uuids and requires enrolment, plus the caller uuid.
func (ctn *QuizCtn) studentParams(c *gin.Context) (courseUuid, contentUuid, userUuid uuid.UUID, ok bool) {
	courseUuid, ok = ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}
	contentUuid, ok = ginutil.ParseUUIDParam(c, "contentUuid")
	if !ok {
		return
	}
	if ok = authutils.CheckCourseAccess(c, courseUuid, ctn.courseService.IsEnrolledOrAdmin, course.ErrCourseNotFound); !ok {
		return
	}
	userUuid, err := authutils.GetUserUUID(c)
	if err != nil {
		ctn.logger.Errorf("Failed to retrieve user UUID from context: %v", err)
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
		return courseUuid, contentUuid, userUuid, false
	}
	return courseUuid, contentUuid, userUuid, true
}

// CreateQuiz godoc
//
//	@Summary		Create draft quiz in a module
//	@Description	Teacher or Admin only. Quiz is a draft until published.
//	@Tags			quiz
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid		path		string				true	"Course UUID"
//	@Param			moduleUuid	path		string				true	"Module UUID"
//	@Param			request		body		CreateQuizRequest	true	"Quiz title"
//	@Success		201			{object}	EditQuizDto
//	@Failure		400			{object}	ginutil.ErrorResponse
//	@Failure		401			{object}	ginutil.ErrorResponse
//	@Failure		403			{object}	ginutil.ErrorResponse
//	@Failure		404			{object}	ginutil.ErrorResponse
//	@Failure		500			{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/module/{moduleUuid}/quiz [post]
func (ctn *QuizCtn) create(c *gin.Context) {
	courseUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}
	moduleUuid, ok := ginutil.ParseUUIDParam(c, "moduleUuid")
	if !ok {
		return
	}
	if !authutils.CheckCourseAccess(c, courseUuid, ctn.courseService.IsTeacherOrAdmin, course.ErrCourseNotFound) {
		return
	}
	var req CreateQuizRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ctn.logger.Errorf("Failed to bind CreateQuiz payload: %v", err)
		ginutil.RespondWithError(c, http.StatusBadRequest, "Invalid request body")
		return
	}
	dto, err := ctn.quizService.CreateQuiz(courseUuid, moduleUuid, req.Title)
	if ctn.handleError(c, err) {
		return
	}
	c.JSON(http.StatusCreated, dto)
}

// GetQuizForEdit godoc
//
//	@Summary		Get full quiz including answer key
//	@Description	Teacher or Admin only.
//	@Tags			quiz
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid		path		string	true	"Course UUID"
//	@Param			contentUuid	path		string	true	"Quiz UUID"
//	@Success		200			{object}	EditQuizDto
//	@Failure		401			{object}	ginutil.ErrorResponse
//	@Failure		403			{object}	ginutil.ErrorResponse
//	@Failure		404			{object}	ginutil.ErrorResponse
//	@Failure		500			{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/quiz/{contentUuid} [get]
func (ctn *QuizCtn) getForEdit(c *gin.Context) {
	courseUuid, contentUuid, ok := ctn.teacherParams(c)
	if !ok {
		return
	}
	dto, err := ctn.quizService.GetEditQuiz(courseUuid, contentUuid)
	if ctn.handleError(c, err) {
		return
	}
	c.JSON(http.StatusOK, dto)
}

// UpdateQuiz godoc
//
//	@Summary		Update quiz settings
//	@Description	Teacher or Admin only. Resets every attempt so students must retake.
//	@Tags			quiz
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid		path		string				true	"Course UUID"
//	@Param			contentUuid	path		string				true	"Quiz UUID"
//	@Param			request		body		UpdateQuizRequest	true	"Quiz settings"
//	@Success		200			{object}	EditQuizDto
//	@Failure		400			{object}	ginutil.ErrorResponse
//	@Failure		401			{object}	ginutil.ErrorResponse
//	@Failure		403			{object}	ginutil.ErrorResponse
//	@Failure		404			{object}	ginutil.ErrorResponse
//	@Failure		500			{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/quiz/{contentUuid} [put]
func (ctn *QuizCtn) update(c *gin.Context) {
	courseUuid, contentUuid, ok := ctn.teacherParams(c)
	if !ok {
		return
	}
	var req UpdateQuizRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ctn.logger.Errorf("Failed to bind UpdateQuiz payload: %v", err)
		ginutil.RespondWithError(c, http.StatusBadRequest, "Invalid request body")
		return
	}
	dto, err := ctn.quizService.UpdateQuiz(courseUuid, contentUuid, req)
	if ctn.handleError(c, err) {
		return
	}
	c.JSON(http.StatusOK, dto)
}

// DeleteQuiz godoc
//
//	@Summary		Delete quiz
//	@Description	Teacher or Admin only. Deletes questions and all attempts.
//	@Tags			quiz
//	@Security		BearerAuth
//	@Param			uuid		path	string	true	"Course UUID"
//	@Param			contentUuid	path	string	true	"Quiz UUID"
//	@Success		204
//	@Failure		401	{object}	ginutil.ErrorResponse
//	@Failure		403	{object}	ginutil.ErrorResponse
//	@Failure		404	{object}	ginutil.ErrorResponse
//	@Failure		500	{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/quiz/{contentUuid} [delete]
func (ctn *QuizCtn) delete(c *gin.Context) {
	courseUuid, contentUuid, ok := ctn.teacherParams(c)
	if !ok {
		return
	}
	if ctn.handleError(c, ctn.quizService.DeleteQuiz(courseUuid, contentUuid)) {
		return
	}
	c.Status(http.StatusNoContent)
}

// ReplaceQuestions godoc
//
//	@Summary		Replace ordered question list
//	@Description	Teacher or Admin only. Array order is question order. Resets every attempt.
//	@Tags			quiz
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid		path		string					true	"Course UUID"
//	@Param			contentUuid	path		string					true	"Quiz UUID"
//	@Param			request		body		ReplaceQuestionsRequest	true	"Questions"
//	@Success		200			{object}	EditQuizDto
//	@Failure		400			{object}	ginutil.ErrorResponse
//	@Failure		401			{object}	ginutil.ErrorResponse
//	@Failure		403			{object}	ginutil.ErrorResponse
//	@Failure		404			{object}	ginutil.ErrorResponse
//	@Failure		500			{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/quiz/{contentUuid}/questions [put]
func (ctn *QuizCtn) replaceQuestions(c *gin.Context) {
	courseUuid, contentUuid, ok := ctn.teacherParams(c)
	if !ok {
		return
	}
	var req ReplaceQuestionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ctn.logger.Errorf("Failed to bind ReplaceQuestions payload: %v", err)
		ginutil.RespondWithError(c, http.StatusBadRequest, "Invalid request body")
		return
	}
	dto, err := ctn.quizService.ReplaceQuestions(courseUuid, contentUuid, req)
	if ctn.handleError(c, err) {
		return
	}
	c.JSON(http.StatusOK, dto)
}

// PublishQuiz godoc
//
//	@Summary		Publish quiz
//	@Description	Teacher or Admin only. Makes quiz visible to students inside its date window.
//	@Tags			quiz
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid		path		string	true	"Course UUID"
//	@Param			contentUuid	path		string	true	"Quiz UUID"
//	@Success		200			{object}	EditQuizDto
//	@Failure		400			{object}	ginutil.ErrorResponse
//	@Failure		401			{object}	ginutil.ErrorResponse
//	@Failure		403			{object}	ginutil.ErrorResponse
//	@Failure		404			{object}	ginutil.ErrorResponse
//	@Failure		500			{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/quiz/{contentUuid}/publish [post]
func (ctn *QuizCtn) publish(c *gin.Context) {
	courseUuid, contentUuid, ok := ctn.teacherParams(c)
	if !ok {
		return
	}
	dto, err := ctn.quizService.Publish(courseUuid, contentUuid)
	if ctn.handleError(c, err) {
		return
	}
	c.JSON(http.StatusOK, dto)
}

// QuizResults godoc
//
//	@Summary		Student results for a quiz
//	@Description	Teacher or Admin only. Best percent and pass/fail per enrolled student.
//	@Tags			quiz
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid		path		string	true	"Course UUID"
//	@Param			contentUuid	path		string	true	"Quiz UUID"
//	@Success		200			{array}		StudentResultRowDto
//	@Failure		401			{object}	ginutil.ErrorResponse
//	@Failure		403			{object}	ginutil.ErrorResponse
//	@Failure		404			{object}	ginutil.ErrorResponse
//	@Failure		500			{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/quiz/{contentUuid}/results [get]
func (ctn *QuizCtn) results(c *gin.Context) {
	courseUuid, contentUuid, ok := ctn.teacherParams(c)
	if !ok {
		return
	}
	rows, err := ctn.quizService.Results(courseUuid, contentUuid)
	if ctn.handleError(c, err) {
		return
	}
	c.JSON(http.StatusOK, rows)
}

// ListAssignments godoc
//
//	@Summary		List course assignments
//	@Description	Quizzes in the course ordered by module. Students see published quizzes and their own best result. Teachers also see drafts.
//	@Tags			quiz
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid	path		string	true	"Course UUID"
//	@Success		200		{array}		AssignmentDto
//	@Failure		401		{object}	ginutil.ErrorResponse
//	@Failure		403		{object}	ginutil.ErrorResponse
//	@Failure		404		{object}	ginutil.ErrorResponse
//	@Failure		500		{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/assignment [get]
func (ctn *QuizCtn) listAssignments(c *gin.Context) {
	courseUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}
	if !authutils.CheckCourseAccess(c, courseUuid, ctn.courseService.IsEnrolledOrAdmin, course.ErrCourseNotFound) {
		return
	}
	userUuid, err := authutils.GetUserUUID(c)
	if err != nil {
		ctn.logger.Errorf("Failed to retrieve user UUID from context: %v", err)
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}
	isTeacher, err := ctn.courseService.IsTeacherOrAdmin(courseUuid, userUuid, authutils.IsAdmin(c))
	if ctn.handleError(c, err) {
		return
	}
	list, err := ctn.quizService.ListAssignments(courseUuid, userUuid, isTeacher)
	if ctn.handleError(c, err) {
		return
	}
	c.JSON(http.StatusOK, list)
}

// StartAttempt godoc
//
//	@Summary		Start or resume attempt
//	@Description	Returns the in-progress attempt, or creates one. Questions never include the answer key.
//	@Tags			quiz
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid		path		string	true	"Course UUID"
//	@Param			contentUuid	path		string	true	"Quiz UUID"
//	@Success		200			{object}	AttemptDto
//	@Failure		401			{object}	ginutil.ErrorResponse
//	@Failure		403			{object}	ginutil.ErrorResponse
//	@Failure		404			{object}	ginutil.ErrorResponse
//	@Failure		409			{object}	ginutil.ErrorResponse
//	@Failure		500			{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/quiz/{contentUuid}/attempt [post]
func (ctn *QuizCtn) startAttempt(c *gin.Context) {
	courseUuid, contentUuid, userUuid, ok := ctn.studentParams(c)
	if !ok {
		return
	}
	dto, err := ctn.quizService.StartAttempt(courseUuid, contentUuid, userUuid)
	if ctn.handleError(c, err) {
		return
	}
	c.JSON(http.StatusOK, dto)
}

// GetAttempt godoc
//
//	@Summary		Get attempt in progress
//	@Description	Questions without answer key plus the answers saved so far.
//	@Tags			quiz
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid		path		string	true	"Course UUID"
//	@Param			contentUuid	path		string	true	"Quiz UUID"
//	@Success		200			{object}	AttemptDto
//	@Failure		401			{object}	ginutil.ErrorResponse
//	@Failure		403			{object}	ginutil.ErrorResponse
//	@Failure		404			{object}	ginutil.ErrorResponse
//	@Failure		500			{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/quiz/{contentUuid}/attempt [get]
func (ctn *QuizCtn) getAttempt(c *gin.Context) {
	courseUuid, contentUuid, userUuid, ok := ctn.studentParams(c)
	if !ok {
		return
	}
	dto, err := ctn.quizService.GetAttempt(courseUuid, contentUuid, userUuid)
	if ctn.handleError(c, err) {
		return
	}
	c.JSON(http.StatusOK, dto)
}

// SaveAnswer godoc
//
//	@Summary		Save answer to one question
//	@Description	Selected holds the marked option indices. Overwrites an earlier answer.
//	@Tags			quiz
//	@Accept			json
//	@Security		BearerAuth
//	@Param			uuid			path	string				true	"Course UUID"
//	@Param			contentUuid		path	string				true	"Quiz UUID"
//	@Param			questionUuid	path	string				true	"Question UUID"
//	@Param			request			body	SaveAnswerRequest	true	"Selected option indices"
//	@Success		204
//	@Failure		400	{object}	ginutil.ErrorResponse
//	@Failure		401	{object}	ginutil.ErrorResponse
//	@Failure		403	{object}	ginutil.ErrorResponse
//	@Failure		404	{object}	ginutil.ErrorResponse
//	@Failure		500	{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/quiz/{contentUuid}/attempt/answers/{questionUuid} [put]
func (ctn *QuizCtn) saveAnswer(c *gin.Context) {
	courseUuid, contentUuid, userUuid, ok := ctn.studentParams(c)
	if !ok {
		return
	}
	questionUuid, ok := ginutil.ParseUUIDParam(c, "questionUuid")
	if !ok {
		return
	}
	var req SaveAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ctn.logger.Errorf("Failed to bind SaveAnswer payload: %v", err)
		ginutil.RespondWithError(c, http.StatusBadRequest, "Invalid request body")
		return
	}
	err := ctn.quizService.SaveAnswer(courseUuid, contentUuid, userUuid, questionUuid, req.Selected)
	if ctn.handleError(c, err) {
		return
	}
	c.Status(http.StatusNoContent)
}

// SubmitAttempt godoc
//
//	@Summary		Submit attempt
//	@Description	Grades on the server. Unanswered questions score 0. Returns the percent and pass/fail of the attempt just submitted, never which answers were right. The best attempt still counts in the assignment list.
//	@Tags			quiz
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid		path		string	true	"Course UUID"
//	@Param			contentUuid	path		string	true	"Quiz UUID"
//	@Success		200			{object}	ResultDto
//	@Failure		400			{object}	ginutil.ErrorResponse
//	@Failure		401			{object}	ginutil.ErrorResponse
//	@Failure		403			{object}	ginutil.ErrorResponse
//	@Failure		404			{object}	ginutil.ErrorResponse
//	@Failure		500			{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/quiz/{contentUuid}/attempt/submit [post]
func (ctn *QuizCtn) submit(c *gin.Context) {
	courseUuid, contentUuid, userUuid, ok := ctn.studentParams(c)
	if !ok {
		return
	}
	result, err := ctn.quizService.SubmitAttempt(courseUuid, contentUuid, userUuid)
	if ctn.handleError(c, err) {
		return
	}
	c.JSON(http.StatusOK, result)
}
