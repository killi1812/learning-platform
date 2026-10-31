package submission

import (
	"errors"
	"net/http"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/ginutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// SubmissionCtn provides HTTP endpoints for assignment submissions.
type SubmissionCtn struct {
	Service   ISubmissionService
	Publisher ISubmissionPublisher
	logger    *zap.SugaredLogger
}

// NewSubmissionCtn constructs a new SubmissionCtn as an app.Api.
func NewSubmissionCtn(service ISubmissionService, publisher ISubmissionPublisher, logger *zap.SugaredLogger) app.Api {
	return &SubmissionCtn{
		Service:   service,
		Publisher: publisher,
		logger:    logger,
	}
}

// RegisterEndpoints mounts submission routes onto the Gin router group.
func (ctn *SubmissionCtn) RegisterEndpoints(api *gin.RouterGroup) {
	courses := api.Group("/course")
	courses.Use(authutils.Protect())

	courses.POST("/:uuid/content/:contentUuid/submission", ctn.submit)
}

func (ctn *SubmissionCtn) handleError(c *gin.Context, err error, fallbackMsg string) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, ErrContentNotFound):
		ginutil.RespondWithError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrNotAssignment),
		errors.Is(err, ErrAssignmentTypeUnsupported),
		errors.Is(err, ErrDeadlinePassed):
		ginutil.RespondWithError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrNotEnrolled),
		errors.Is(err, ErrModuleLocked):
		ginutil.RespondWithError(c, http.StatusForbidden, err.Error())
	default:
		ctn.logger.Errorf("%s: %v", fallbackMsg, err)
		ginutil.RespondWithError(c, http.StatusInternalServerError, fallbackMsg)
	}
	return true
}

// SubmitAssignment godoc
//
//	@Summary		Submit assignment deliverable
//	@Description	Stamps receipt time, verifies eligibility, and enqueues the submission for asynchronous persistence.
//	@Tags			submission
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid		path		string				true	"Course UUID"
//	@Param			contentUuid	path		string				true	"Content UUID"
//	@Param			submission	body		CreateSubmissionDto	true	"Submission payload"
//	@Success		202			{object}	SubmissionAcceptedDto
//	@Failure		400			{object}	ginutil.ErrorResponse
//	@Failure		401			{object}	ginutil.ErrorResponse
//	@Failure		403			{object}	ginutil.ErrorResponse
//	@Failure		404			{object}	ginutil.ErrorResponse
//	@Failure		500			{object}	ginutil.ErrorResponse
//	@Failure		503			{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/content/{contentUuid}/submission [post]
func (ctn *SubmissionCtn) submit(c *gin.Context) {
	// Stamp receive time as the first action upon entering backend
	submittedAt := time.Now().UTC()

	courseUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}

	contentUuid, ok := ginutil.ParseUUIDParam(c, "contentUuid")
	if !ok {
		return
	}

	userUuid, err := authutils.GetUserUUID(c)
	if err != nil {
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var dto CreateSubmissionDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		ginutil.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	if dto.PayloadType == PayloadTypeFile {
		ginutil.RespondWithError(c, http.StatusBadRequest, "payload type FILE is not supported yet")
		return
	}

	target, err := ctn.Service.ResolveTarget(courseUuid, contentUuid, userUuid, submittedAt)
	if ctn.handleError(c, err, "Failed to resolve submission target") {
		return
	}

	subUuid := uuid.New()
	msg := SubmissionMessage{
		Uuid:        subUuid,
		ContentID:   target.ContentID,
		UserID:      target.UserID,
		UserEmail:   target.UserEmail,
		PayloadType: dto.PayloadType,
		PayloadURL:  dto.PayloadURL,
		SubmittedAt: submittedAt,
	}

	if err := ctn.Publisher.Publish(c.Request.Context(), msg); err != nil {
		ctn.logger.Errorf("Failed to publish submission %s to queue: %v", subUuid, err)
		ginutil.RespondWithError(c, http.StatusServiceUnavailable, "Failed to queue submission")
		return
	}

	c.JSON(http.StatusAccepted, SubmissionAcceptedDto{
		Uuid:        subUuid,
		SubmittedAt: submittedAt,
		Status:      "QUEUED",
	})
}
