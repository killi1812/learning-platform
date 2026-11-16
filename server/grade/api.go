package grade

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/ginutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// GradeCtn provides HTTP endpoints for grading submissions and accessing gradebooks.
type GradeCtn struct {
	Service       IGradeService
	courseService course.ICourseService
	logger        *zap.SugaredLogger
}

// NewGradeCtn constructs a new GradeCtn as an app.Api.
func NewGradeCtn(
	service IGradeService,
	courseService course.ICourseService,
	logger *zap.SugaredLogger,
) app.Api {
	return &GradeCtn{
		Service:       service,
		courseService: courseService,
		logger:        logger,
	}
}

// RegisterEndpoints mounts grading routes onto the Gin router group.
func (ctn *GradeCtn) RegisterEndpoints(api *gin.RouterGroup) {
	courses := api.Group("/course")
	courses.Use(authutils.Protect())

	courses.PUT("/:uuid/content/:contentUuid/submission/:submissionUuid/grade", ctn.upsertGrade)
	courses.POST("/:uuid/content/:contentUuid/grade/release", ctn.releaseGrades)
	courses.GET("/:uuid/grade/export", ctn.exportCSV)
	courses.GET("/:uuid/grade/roster", ctn.getCourseRoster)

	grades := api.Group("/grade")
	grades.Use(authutils.Protect())

	grades.GET("/pending", ctn.listPending)
	grades.GET("/my", ctn.listMyGrades)
	grades.GET("/my/final", ctn.getMyFinalGrades)
}

func (ctn *GradeCtn) handleError(c *gin.Context, err error, fallbackMsg string) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, ErrSubmissionNotFound),
		errors.Is(err, ErrContentNotFound),
		errors.Is(err, course.ErrCourseNotFound):
		ctn.logger.Errorf("Resource not found: %v", err)
		ginutil.RespondWithError(c, http.StatusNotFound, err.Error())
		return true
	case errors.Is(err, ErrGraderNotFound):
		ctn.logger.Errorf("Grader not found: %v", err)
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized grader")
		return true
	case errors.Is(err, ErrUserNotFound):
		ctn.logger.Errorf("User not found: %v", err)
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
		return true
	case errors.Is(err, ErrNotTeacher):
		ctn.logger.Errorf("Teacher access required: %v", err)
		ginutil.RespondWithError(c, http.StatusForbidden, "Forbidden")
		return true
	case errors.Is(err, ErrNotHandin):
		ctn.logger.Errorf("Invalid content type for grading: %v", err)
		ginutil.RespondWithError(c, http.StatusConflict, err.Error())
		return true
	case errors.Is(err, ErrScoreOutOfRange):
		ctn.logger.Errorf("Grading domain validation error: %v", err)
		ginutil.RespondWithError(c, http.StatusUnprocessableEntity, err.Error())
		return true
	default:
		ctn.logger.Errorf("%s: %v", fallbackMsg, err)
		ginutil.RespondWithError(c, http.StatusInternalServerError, "Internal server error")
		return true
	}
}

// UpsertGrade godoc
//
//	@Summary		Grade or update an individual or group submission
//	@Description	Teacher or Admin only. Sets evaluation score, Markdown feedback, and student visibility.
//	@Tags			grade
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid			path		string			true	"Course UUID"
//	@Param			contentUuid		path		string			true	"Content UUID"
//	@Param			submissionUuid	path		string			true	"Submission UUID"
//	@Param			grade			body		UpsertGradeDto	true	"Grade and feedback payload"
//	@Success		200				{object}	GradeDto
//	@Failure		400				{object}	ginutil.ErrorResponse
//	@Failure		401				{object}	ginutil.ErrorResponse
//	@Failure		403				{object}	ginutil.ErrorResponse
//	@Failure		404				{object}	ginutil.ErrorResponse
//	@Failure		409				{object}	ginutil.ErrorResponse
//	@Failure		422				{object}	ginutil.ErrorResponse
//	@Failure		500				{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/content/{contentUuid}/submission/{submissionUuid}/grade [put]
func (ctn *GradeCtn) upsertGrade(c *gin.Context) {
	courseUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}
	contentUuid, ok := ginutil.ParseUUIDParam(c, "contentUuid")
	if !ok {
		return
	}
	submissionUuid, ok := ginutil.ParseUUIDParam(c, "submissionUuid")
	if !ok {
		return
	}

	if !authutils.CheckCourseAccess(c, courseUuid, ctn.courseService.IsTeacherOrAdmin, course.ErrCourseNotFound) {
		return
	}

	graderUuid, err := authutils.GetUserUUID(c)
	if err != nil {
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var dto UpsertGradeDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		ctn.logger.Errorf("Failed to bind UpsertGrade payload: %v", err)
		ginutil.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	res, err := ctn.Service.UpsertGrade(courseUuid, contentUuid, submissionUuid, graderUuid, dto)
	if ctn.handleError(c, err, "Failed to upsert grade") {
		return
	}

	c.JSON(http.StatusOK, res)
}

// ReleaseGrades godoc
//
//	@Summary		Release all unreleased grades for an assignment
//	@Description	Teacher or Admin only. Transitions all unreleased individual or group grades to visible and notifies students.
//	@Tags			grade
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid		path		string	true	"Course UUID"
//	@Param			contentUuid	path		string	true	"Content UUID"
//	@Success		200			{object}	ReleaseResultDto
//	@Failure		400			{object}	ginutil.ErrorResponse
//	@Failure		401			{object}	ginutil.ErrorResponse
//	@Failure		403			{object}	ginutil.ErrorResponse
//	@Failure		404			{object}	ginutil.ErrorResponse
//	@Failure		409			{object}	ginutil.ErrorResponse
//	@Failure		422			{object}	ginutil.ErrorResponse
//	@Failure		500			{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/content/{contentUuid}/grade/release [post]
func (ctn *GradeCtn) releaseGrades(c *gin.Context) {
	courseUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}
	contentUuid, ok := ginutil.ParseUUIDParam(c, "contentUuid")
	if !ok {
		return
	}

	if !authutils.CheckCourseAccess(c, courseUuid, ctn.courseService.IsTeacherOrAdmin, course.ErrCourseNotFound) {
		return
	}

	count, err := ctn.Service.ReleaseAssignmentGrades(courseUuid, contentUuid)
	if ctn.handleError(c, err, "Failed to release grades") {
		return
	}

	c.JSON(http.StatusOK, ReleaseResultDto{Released: count})
}

// ListPending godoc
//
//	@Summary		List pending ungraded submissions
//	@Description	Teacher or Admin only. Returns latest ungraded individual and group submissions across courses taught by caller.
//	@Tags			grade
//	@Produce		json
//	@Security		BearerAuth
//	@Param			courseUuid	query		string	false	"Filter by course UUID"
//	@Success		200			{array}		PendingSubmissionDto
//	@Failure		400			{object}	ginutil.ErrorResponse
//	@Failure		401			{object}	ginutil.ErrorResponse
//	@Failure		403			{object}	ginutil.ErrorResponse
//	@Failure		404			{object}	ginutil.ErrorResponse
//	@Failure		500			{object}	ginutil.ErrorResponse
//	@Router			/grade/pending [get]
func (ctn *GradeCtn) listPending(c *gin.Context) {
	userUuid, err := authutils.GetUserUUID(c)
	if err != nil {
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var courseUuidPtr *uuid.UUID
	if cStr := c.Query("courseUuid"); cStr != "" {
		parsed, err := uuid.Parse(cStr)
		if err != nil {
			ginutil.RespondWithError(c, http.StatusBadRequest, "Invalid course UUID parameter")
			return
		}
		courseUuidPtr = &parsed
	}

	isAdmin := authutils.IsAdmin(c)
	pending, err := ctn.Service.ListPending(userUuid, isAdmin, courseUuidPtr)
	if ctn.handleError(c, err, "Failed to list pending submissions") {
		return
	}

	c.JSON(http.StatusOK, pending)
}

// ListMyGrades godoc
//
//	@Summary		List student's own grades
//	@Description	Student view. Returns released hand-in grades and quiz best-attempt scores for authenticated student.
//	@Tags			grade
//	@Produce		json
//	@Security		BearerAuth
//	@Param			courseUuid	query		string	false	"Filter by course UUID"
//	@Success		200			{array}		StudentGradeDto
//	@Failure		400			{object}	ginutil.ErrorResponse
//	@Failure		401			{object}	ginutil.ErrorResponse
//	@Failure		500			{object}	ginutil.ErrorResponse
//	@Router			/grade/my [get]
func (ctn *GradeCtn) listMyGrades(c *gin.Context) {
	userUuid, err := authutils.GetUserUUID(c)
	if err != nil {
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var courseUuidPtr *uuid.UUID
	if cStr := c.Query("courseUuid"); cStr != "" {
		parsed, err := uuid.Parse(cStr)
		if err != nil {
			ginutil.RespondWithError(c, http.StatusBadRequest, "Invalid course UUID parameter")
			return
		}
		courseUuidPtr = &parsed
	}

	grades, err := ctn.Service.ListMyGrades(userUuid, courseUuidPtr)
	if ctn.handleError(c, err, "Failed to list student grades") {
		return
	}

	c.JSON(http.StatusOK, grades)
}

// ExportCSV godoc
//
//	@Summary		Export course gradebook as CSV
//	@Description	Teacher or Admin only. Streams CSV containing all student grades across all assignments in course.
//	@Tags			grade
//	@Produce		text/csv
//	@Security		BearerAuth
//	@Param			uuid	path	string	true	"Course UUID"
//	@Success		200
//	@Failure		400	{object}	ginutil.ErrorResponse
//	@Failure		401	{object}	ginutil.ErrorResponse
//	@Failure		403	{object}	ginutil.ErrorResponse
//	@Failure		404	{object}	ginutil.ErrorResponse
//	@Failure		500	{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/grade/export [get]
func (ctn *GradeCtn) exportCSV(c *gin.Context) {
	courseUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}

	if !authutils.CheckCourseAccess(c, courseUuid, ctn.courseService.IsTeacherOrAdmin, course.ErrCourseNotFound) {
		return
	}

	var buf bytes.Buffer
	if err := ctn.Service.ExportCourseCSV(courseUuid, &buf); err != nil {
		ctn.handleError(c, err, "Failed to export gradebook CSV")
		return
	}

	filename := fmt.Sprintf("course-%s-gradebook-%s.csv", courseUuid.String()[:8], time.Now().Format("20060102"))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, "text/csv; charset=utf-8", buf.Bytes())
}

// GetMyFinalGrades godoc
//
//	@Summary		Get student's final course grades
//	@Description	Calculates and returns cumulative final subject letter grades (A-F), earned points, and overall percentage across enrolled courses.
//	@Tags			grade
//	@Produce		json
//	@Security		BearerAuth
//	@Param			courseUuid	query		string	false	"Filter by course UUID"
//	@Success		200			{array}		CourseFinalGradeDto
//	@Failure		400			{object}	ginutil.ErrorResponse
//	@Failure		401			{object}	ginutil.ErrorResponse
//	@Failure		500			{object}	ginutil.ErrorResponse
//	@Router			/grade/my/final [get]
func (ctn *GradeCtn) getMyFinalGrades(c *gin.Context) {
	userUuid, err := authutils.GetUserUUID(c)
	if err != nil {
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var courseUuidPtr *uuid.UUID
	if cStr := c.Query("courseUuid"); cStr != "" {
		parsed, err := uuid.Parse(cStr)
		if err != nil {
			ginutil.RespondWithError(c, http.StatusBadRequest, "Invalid course UUID parameter")
			return
		}
		courseUuidPtr = &parsed
	}

	finals, err := ctn.Service.GetStudentFinalGrades(userUuid, courseUuidPtr)
	if ctn.handleError(c, err, "Failed to retrieve student final grades") {
		return
	}

	c.JSON(http.StatusOK, finals)
}

// GetCourseRoster godoc
//
//	@Summary		Get course grade roster with final letter grades
//	@Description	Teacher or Admin only. Returns all enrolled students with their cumulative earned scores, percentages, and final letter grades (A-F).
//	@Tags			grade
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid	path		string	true	"Course UUID"
//	@Success		200		{array}		StudentFinalGradeRowDto
//	@Failure		400		{object}	ginutil.ErrorResponse
//	@Failure		401		{object}	ginutil.ErrorResponse
//	@Failure		403		{object}	ginutil.ErrorResponse
//	@Failure		404		{object}	ginutil.ErrorResponse
//	@Failure		500		{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/grade/roster [get]
func (ctn *GradeCtn) getCourseRoster(c *gin.Context) {
	courseUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}

	if !authutils.CheckCourseAccess(c, courseUuid, ctn.courseService.IsTeacherOrAdmin, course.ErrCourseNotFound) {
		return
	}

	roster, err := ctn.Service.GetCourseRosterFinalGrades(courseUuid)
	if ctn.handleError(c, err, "Failed to retrieve course grade roster") {
		return
	}

	c.JSON(http.StatusOK, roster)
}
