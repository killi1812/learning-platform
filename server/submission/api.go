package submission

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/ginutil"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/storage"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// SubmissionCtn provides HTTP endpoints for assignment submissions.
type SubmissionCtn struct {
	Service        ISubmissionService
	Publisher      ISubmissionPublisher
	storageService storage.IStorageService
	courseService  course.ICourseService
	logger         *zap.SugaredLogger
}

// NewSubmissionCtn constructs a new SubmissionCtn as an app.Api.
func NewSubmissionCtn(
	service ISubmissionService,
	publisher ISubmissionPublisher,
	storageService storage.IStorageService,
	courseService course.ICourseService,
	logger *zap.SugaredLogger,
) app.Api {
	return &SubmissionCtn{
		Service:        service,
		Publisher:      publisher,
		storageService: storageService,
		courseService:  courseService,
		logger:         logger,
	}
}

// RegisterEndpoints mounts submission routes onto the Gin router group.
func (ctn *SubmissionCtn) RegisterEndpoints(api *gin.RouterGroup) {
	courses := api.Group("/course")
	courses.Use(authutils.Protect())

	courses.POST("/:uuid/content/:contentUuid/submission", ctn.submit)
	courses.POST("/:uuid/content/:contentUuid/submission/file/init", ctn.initFileSubmission)
	courses.POST("/:uuid/content/:contentUuid/submission/file/upload", ctn.uploadFileSubmission)
	courses.GET("/:uuid/content/:contentUuid/submission/my", ctn.listMySubmissions)
	courses.GET("/:uuid/content/:contentUuid/submission", ctn.listCourseSubmissions)
	courses.GET("/:uuid/content/:contentUuid/submission/:submissionUuid/file", ctn.downloadSubmissionFile)
}

func (ctn *SubmissionCtn) handleError(c *gin.Context, err error, fallbackMsg string) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, ErrContentNotFound),
		errors.Is(err, ErrSubmissionNotFound),
		errors.Is(err, ErrTicketNotFound),
		errors.Is(err, storage.ErrObjectNotFound):
		ctn.logger.Errorf("%s (not found): %v", fallbackMsg, err)
		ginutil.RespondWithError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrNotAssignment),
		errors.Is(err, ErrAssignmentTypeUnsupported),
		errors.Is(err, ErrDeadlinePassed),
		errors.Is(err, ErrTicketExpired),
		errors.Is(err, ErrTicketConsumed):
		ctn.logger.Errorf("%s (bad request): %v", fallbackMsg, err)
		ginutil.RespondWithError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrNotEnrolled),
		errors.Is(err, ErrModuleLocked),
		errors.Is(err, ErrUnauthorizedSubmission),
		errors.Is(err, ErrNoGroup),
		errors.Is(err, ErrGroupChanged):
		ctn.logger.Errorf("%s (forbidden): %v", fallbackMsg, err)
		ginutil.RespondWithError(c, http.StatusForbidden, err.Error())
	default:
		ctn.logger.Errorf("%s: %v", fallbackMsg, err)
		ginutil.RespondWithError(c, http.StatusInternalServerError, fallbackMsg)
	}
	return true
}

// SubmitAssignment godoc
//
//	@Summary		Submit assignment deliverable via URL
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
		ctn.logger.Errorf("Failed to retrieve user UUID from context: %v", err)
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var dto CreateSubmissionDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		ctn.logger.Errorf("Failed to bind CreateSubmission payload: %v", err)
		ginutil.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	if dto.PayloadType == PayloadTypeFile {
		ctn.logger.Errorf("Direct file payload unsupported on /submission, use /submission/file/init")
		ginutil.RespondWithError(c, http.StatusBadRequest, "payload type FILE is not supported yet on this endpoint, use /submission/file/init")
		return
	}

	target, err := ctn.Service.ResolveTarget(courseUuid, contentUuid, userUuid, submittedAt)
	if ctn.handleError(c, err, "Failed to resolve submission target") {
		return
	}

	subUuid := uuid.New()
	msg := SubmissionMessage{
		Uuid:         subUuid,
		ContentID:    target.ContentID,
		ContentTitle: target.ContentTitle,
		CourseUuid:   target.CourseUuid,
		UserID:       target.UserID,
		UserUuid:     target.UserUuid,
		UserEmail:    target.UserEmail,
		GroupID:      target.GroupID,
		GroupName:    target.GroupName,
		Members:      target.Members,
		PayloadType:  dto.PayloadType,
		PayloadURL:   dto.PayloadURL,
		SubmittedAt:  submittedAt,
	}

	if err := ctn.Publisher.Publish(c.Request.Context(), msg); err != nil {
		ctn.logger.Errorf("Failed to publish submission %s to queue: %v", subUuid, err)
		ctn.Service.NotifySubmissionFailed(userUuid, courseUuid, target.ContentTitle)
		ginutil.RespondWithError(c, http.StatusServiceUnavailable, "Failed to queue submission")
		return
	}

	ctn.Service.NotifySubmitted(target)

	c.JSON(http.StatusAccepted, SubmissionAcceptedDto{
		Uuid:        subUuid,
		SubmittedAt: submittedAt,
		Status:      "QUEUED",
	})
}

// InitFileSubmission godoc
//
//	@Summary		Initialize file submission
//	@Description	Stamps receipt time, verifies eligibility, and issues an upload ticket valid for 30 minutes.
//	@Tags			submission
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid		path		string	true	"Course UUID"
//	@Param			contentUuid	path		string	true	"Content UUID"
//	@Success		201			{object}	UploadTicketDto
//	@Failure		400			{object}	ginutil.ErrorResponse
//	@Failure		401			{object}	ginutil.ErrorResponse
//	@Failure		403			{object}	ginutil.ErrorResponse
//	@Failure		404			{object}	ginutil.ErrorResponse
//	@Failure		500			{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/content/{contentUuid}/submission/file/init [post]
func (ctn *SubmissionCtn) initFileSubmission(c *gin.Context) {
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
		ctn.logger.Errorf("Failed to retrieve user UUID from context: %v", err)
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	ticket, err := ctn.Service.InitFileSubmission(courseUuid, contentUuid, userUuid, submittedAt)
	if ctn.handleError(c, err, "Failed to initialize file submission") {
		return
	}

	c.JSON(http.StatusCreated, UploadTicketDto{
		TicketUuid:  ticket.Uuid,
		SubmittedAt: ticket.SubmittedAt,
		ExpiresAt:   ticket.ExpiresAt,
	})
}

type activityReader struct {
	reader    io.Reader
	rc        *http.ResponseController
	bytesRead int64
	maxBytes  int64
	overflow  bool
}

func (a *activityReader) Read(p []byte) (int, error) {
	n, err := a.reader.Read(p)
	if n > 0 {
		a.bytesRead += int64(n)
		if a.bytesRead > a.maxBytes {
			a.overflow = true
			return n, errors.New("file size limit exceeded")
		}
		if a.rc != nil {
			_ = a.rc.SetReadDeadline(time.Now().Add(30 * time.Second))
			_ = a.rc.SetWriteDeadline(time.Now().Add(30 * time.Second))
		}
	}
	return n, err
}

// UploadFileSubmission godoc
//
//	@Summary		Stream upload deliverable file for submission ticket
//	@Description	Accepts multipart file stream, pipes zero-memory to SeaweedFS with rolling deadline extension, and queues submission.
//	@Tags			submission
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid		path		string	true	"Course UUID"
//	@Param			contentUuid	path		string	true	"Content UUID"
//	@Param			ticket		query		string	true	"Upload Ticket UUID"
//	@Param			file		formData	file	true	"Deliverable file"
//	@Success		202			{object}	SubmissionAcceptedDto
//	@Failure		400			{object}	ginutil.ErrorResponse
//	@Failure		401			{object}	ginutil.ErrorResponse
//	@Failure		403			{object}	ginutil.ErrorResponse
//	@Failure		404			{object}	ginutil.ErrorResponse
//	@Failure		500			{object}	ginutil.ErrorResponse
//	@Failure		503			{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/content/{contentUuid}/submission/file/upload [post]
func (ctn *SubmissionCtn) uploadFileSubmission(c *gin.Context) {
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
		ctn.logger.Errorf("Failed to retrieve user UUID from context: %v", err)
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	ticketStr := c.Query("ticket")
	if ticketStr == "" {
		ticketStr = c.GetHeader("X-Upload-Ticket")
	}
	ticketUuid, err := uuid.Parse(ticketStr)
	if err != nil {
		ctn.logger.Errorf("Failed to parse upload ticket UUID %q: %v", ticketStr, err)
		ginutil.RespondWithError(c, http.StatusBadRequest, "Invalid upload ticket")
		return
	}

	ticket, err := ctn.Service.ValidateAndExtendTicket(ticketUuid, courseUuid, contentUuid, userUuid, time.Now().UTC())
	if ctn.handleError(c, err, "Invalid upload ticket") {
		return
	}

	// 200MB limit
	const maxUploadSize = 200 * 1024 * 1024
	rc := http.NewResponseController(c.Writer)
	_ = rc.SetReadDeadline(time.Now().Add(30 * time.Second))
	_ = rc.SetWriteDeadline(time.Now().Add(30 * time.Second))

	mr, err := c.Request.MultipartReader()
	if err != nil {
		ctn.logger.Errorf("Failed to open multipart reader for upload: %v", err)
		ginutil.RespondWithError(c, http.StatusBadRequest, "Invalid multipart request")
		return
	}

	var fileName string
	var contentType string
	var storageKey string
	var bytesRead int64

	// TODO: extract to const
	allowedExts := map[string]string{
		".pdf":    "application/pdf",
		".docx":   "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".pptx":   "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		".txt":    "text/plain",
		".md":     "text/plain",
		".png":    "image/png",
		".jpg":    "image/jpeg",
		".jpeg":   "image/jpeg",
		".gif":    "image/gif",
		".webp":   "image/webp",
		".json":   "application/json",
		".zip":    "application/zip",
		".tar.gz": "application/gzip",
		".tgz":    "application/gzip",
	}

	fileFound := false
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			ctn.logger.Errorf("Failed to read multipart stream: %v", err)
			ginutil.RespondWithError(c, http.StatusBadRequest, "Failed to read multipart stream: "+err.Error())
			return
		}

		if part.FormName() == "file" {
			fileFound = true
			rawName := part.FileName()
			if rawName == "" {
				part.Close()
				ctn.logger.Errorf("Upload rejected: filename is empty")
				ginutil.RespondWithError(c, http.StatusBadRequest, "Filename cannot be empty")
				return
			}

			lowerName := strings.ToLower(rawName)
			ext := strings.ToLower(filepath.Ext(lowerName))
			if strings.HasSuffix(lowerName, ".tar.gz") {
				ext = ".tar.gz"
			}

			expectedType, allowed := allowedExts[ext]
			if !allowed {
				part.Close()
				ctn.logger.Errorf("Upload rejected: unsupported file extension %s", ext)
				ginutil.RespondWithError(c, http.StatusBadRequest, "Unsupported file extension")
				return
			}

			cleanName := filepath.Base(filepath.Clean(rawName))
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
			contentType = expectedType
			storageKey = fmt.Sprintf("submissions/%s/%s/%s", contentUuid.String(), ticket.Uuid.String(), cleanName)

			ar := &activityReader{
				reader:   part,
				rc:       rc,
				maxBytes: maxUploadSize,
			}

			uploadErr := ctn.storageService.Upload(storageKey, ar, contentType)
			part.Close()
			if ar.overflow {
				_ = ctn.storageService.Delete(storageKey)
				ctn.logger.Errorf("Upload rejected: file size exceeds 200MB limit")
				ginutil.RespondWithError(c, http.StatusBadRequest, "File size exceeds 200MB limit")
				return
			}
			if uploadErr != nil {
				_ = ctn.storageService.Delete(storageKey)
				ctn.logger.Errorf("Failed to stream file to storage: %v", uploadErr)
				ginutil.RespondWithError(c, http.StatusInternalServerError, "Failed to save uploaded file")
				return
			}

			bytesRead = ar.bytesRead
			break
		}
		part.Close()
	}

	if !fileFound {
		ctn.logger.Errorf("Upload rejected: no file provided in form field 'file'")
		ginutil.RespondWithError(c, http.StatusBadRequest, "No file provided in form field 'file'")
		return
	}

	var groupName string
	var groupMembers []GroupMember
	if ticket.GroupID != nil {
		var gErr error
		groupName, groupMembers, gErr = ctn.Service.ResolveGroupMembers(*ticket.GroupID)
		if gErr != nil {
			_ = ctn.storageService.Delete(storageKey)
			ctn.handleError(c, gErr, "Failed to resolve group members for upload")
			return
		}
	}

	msg := SubmissionMessage{
		Uuid:         ticket.Uuid,
		ContentID:    ticket.ContentID,
		ContentTitle: ticket.ContentTitle,
		CourseUuid:   courseUuid,
		UserID:       ticket.UserID,
		UserUuid:     userUuid,
		UserEmail:    ticket.UserEmail,
		GroupID:      ticket.GroupID,
		GroupName:    groupName,
		Members:      groupMembers,
		PayloadType:  PayloadTypeFile,
		FileName:     fileName,
		FileSize:     bytesRead,
		ContentType:  contentType,
		StorageKey:   storageKey,
		SubmittedAt:  ticket.SubmittedAt,
	}

	if err := ctn.Service.ConsumeUploadTicket(ticket.Uuid); err != nil {
		_ = ctn.storageService.Delete(storageKey)
		ctn.handleError(c, err, "Failed to consume upload ticket")
		return
	}

	if err := ctn.Publisher.Publish(c.Request.Context(), msg); err != nil {
		ctn.logger.Errorf("Failed to publish submission %s to queue: %v", ticket.Uuid, err)
		_ = ctn.storageService.Delete(storageKey)
		ctn.Service.NotifySubmissionFailed(userUuid, courseUuid, ticket.ContentTitle)
		ginutil.RespondWithError(c, http.StatusServiceUnavailable, "Failed to queue submission")
		return
	}

	ctn.Service.NotifySubmitted(&SubmissionTarget{
		CourseID:     ticket.CourseID,
		ContentID:    ticket.ContentID,
		ContentTitle: ticket.ContentTitle,
		CourseUuid:   courseUuid,
		UserID:       ticket.UserID,
		UserUuid:     userUuid,
		UserEmail:    ticket.UserEmail,
		GroupID:      ticket.GroupID,
		GroupName:    groupName,
		Members:      groupMembers,
	})

	c.JSON(http.StatusAccepted, SubmissionAcceptedDto{
		Uuid:        ticket.Uuid,
		SubmittedAt: ticket.SubmittedAt,
		Status:      "QUEUED", // TODO: extract to type and  consts
	})
}

// ListMySubmissions godoc
//
//	@Summary		List my submissions for an assignment
//	@Description	Returns all submission attempts made by the authenticated student for this assignment.
//	@Tags			submission
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid		path		string	true	"Course UUID"
//	@Param			contentUuid	path		string	true	"Content UUID"
//	@Success		200			{array}		SubmissionDto
//	@Failure		400			{object}	ginutil.ErrorResponse
//	@Failure		401			{object}	ginutil.ErrorResponse
//	@Failure		403			{object}	ginutil.ErrorResponse
//	@Failure		404			{object}	ginutil.ErrorResponse
//	@Failure		500			{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/content/{contentUuid}/submission/my [get]
func (ctn *SubmissionCtn) listMySubmissions(c *gin.Context) {
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
		ctn.logger.Errorf("Failed to retrieve user UUID from context: %v", err)
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	subs, err := ctn.Service.ListMySubmissions(courseUuid, contentUuid, userUuid)
	if ctn.handleError(c, err, "Failed to retrieve student submissions") {
		return
	}

	c.JSON(http.StatusOK, subs)
}

// ListCourseSubmissions godoc
//
//	@Summary		List all student submissions for an assignment
//	@Description	Teacher or Admin only. Returns all student submissions for an assignment.
//	@Tags			submission
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid		path		string	true	"Course UUID"
//	@Param			contentUuid	path		string	true	"Content UUID"
//	@Success		200			{array}		SubmissionDto
//	@Failure		400			{object}	ginutil.ErrorResponse
//	@Failure		401			{object}	ginutil.ErrorResponse
//	@Failure		403			{object}	ginutil.ErrorResponse
//	@Failure		404			{object}	ginutil.ErrorResponse
//	@Failure		500			{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/content/{contentUuid}/submission [get]
func (ctn *SubmissionCtn) listCourseSubmissions(c *gin.Context) {
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

	subs, err := ctn.Service.ListCourseAssignmentSubmissions(courseUuid, contentUuid)
	if ctn.handleError(c, err, "Failed to retrieve course submissions") {
		return
	}

	c.JSON(http.StatusOK, subs)
}

// DownloadSubmissionFile godoc
//
//	@Summary		Download submitted deliverable file
//	@Description	Streams deliverable file for student owner or course instructor.
//	@Tags			submission
//	@Security		BearerAuth
//	@Param			uuid			path	string	true	"Course UUID"
//	@Param			contentUuid		path	string	true	"Content UUID"
//	@Param			submissionUuid	path	string	true	"Submission UUID"
//	@Success		200
//	@Failure		400	{object}	ginutil.ErrorResponse
//	@Failure		401	{object}	ginutil.ErrorResponse
//	@Failure		403	{object}	ginutil.ErrorResponse
//	@Failure		404	{object}	ginutil.ErrorResponse
//	@Failure		500	{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/content/{contentUuid}/submission/{submissionUuid}/file [get]
func (ctn *SubmissionCtn) downloadSubmissionFile(c *gin.Context) {
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

	userUuid, err := authutils.GetUserUUID(c)
	if err != nil {
		ctn.logger.Errorf("Failed to retrieve user UUID from context: %v", err)
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	isPrivileged, _ := ctn.courseService.IsTeacherOrAdmin(courseUuid, userUuid, authutils.IsAdmin(c))
	sub, err := ctn.Service.GetSubmissionForDownload(courseUuid, contentUuid, submissionUuid, userUuid, isPrivileged)
	if ctn.handleError(c, err, "Failed to resolve submission for download") {
		return
	}

	if sub.PayloadType != PayloadTypeFile || sub.StorageKey == "" {
		ctn.logger.Errorf("Download rejected: submission %s deliverable is not a stored file", submissionUuid)
		ginutil.RespondWithError(c, http.StatusBadRequest, "Submission deliverable is not a stored file")
		return
	}

	reader, ct, err := ctn.storageService.Get(sub.StorageKey)
	if err != nil {
		ctn.handleError(c, err, "Failed to stream submission file from storage")
		return
	}
	defer reader.Close()

	cleanName := filepath.Base(sub.FileName)
	if cleanName == "" || cleanName == "." {
		cleanName = "submission_file"
	}
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": cleanName})

	c.Header("Content-Disposition", disposition)
	c.Header("X-Content-Type-Options", "nosniff")

	if ct == "" {
		ct = sub.ContentType
	}
	if ct == "" {
		ct = "application/octet-stream"
	}

	c.DataFromReader(http.StatusOK, sub.FileSize, ct, reader, nil)
}
