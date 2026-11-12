package content

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/ginutil"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/storage"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ContentCtn struct {
	contentService IContentService
	courseService  course.ICourseService
	logger         *zap.SugaredLogger
}

func NewContentCtn(contentService IContentService, courseService course.ICourseService, logger *zap.SugaredLogger) app.Api {
	return &ContentCtn{
		contentService: contentService,
		courseService:  courseService,
		logger:         logger,
	}
}

func (ctn *ContentCtn) RegisterEndpoints(api *gin.RouterGroup) {
	files := api.Group("/course/:uuid/module/:moduleUuid/file")
	files.Use(authutils.Protect())

	files.POST("", ctn.upload)
	files.GET("", ctn.list)
	files.GET("/:fileUuid", ctn.get)
	files.GET("/:fileUuid/preview", ctn.preview)
	files.PUT("/:fileUuid", ctn.update)
	files.DELETE("/:fileUuid", ctn.delete)

	asgns := api.Group("/course/:uuid/module/:moduleUuid/assignment")
	asgns.Use(authutils.Protect())

	asgns.POST("", ctn.createHandin)
	asgns.GET("", ctn.listHandins)
	asgns.GET("/:assignmentUuid", ctn.getHandin)
	asgns.PUT("/:assignmentUuid", ctn.updateHandin)
	asgns.DELETE("/:assignmentUuid", ctn.deleteHandin)
}

func (ctn *ContentCtn) handleError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, ErrContentNotFound),
		errors.Is(err, ErrModuleNotFound),
		errors.Is(err, ErrCourseNotFound),
		errors.Is(err, course.ErrCourseNotFound),
		errors.Is(err, course.ErrModuleNotFound),
		errors.Is(err, storage.ErrObjectNotFound):
		ginutil.RespondWithError(c, http.StatusNotFound, err.Error())
		return true
	case errors.Is(err, ErrModuleLocked),
		errors.Is(err, ErrContentNotAvailable):
		ginutil.RespondWithError(c, http.StatusForbidden, err.Error())
		return true
	case errors.Is(err, ErrGroupModeLocked):
		ginutil.RespondWithError(c, http.StatusConflict, err.Error())
		return true
	default:
		ctn.logger.Errorf("Unexpected content error: %v", err)
		ginutil.RespondWithError(c, http.StatusInternalServerError, "Internal server error")
		return true
	}
}

// UploadFile godoc
//
//	@Summary		Upload or create file material
//	@Description	Uploads course material file (.pdf, .pptx, .docx, .md, .txt) to object storage or creates link (Teacher or Admin only)
//	@Tags			content
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid		path		string	true	"Course UUID"
//	@Param			moduleUuid	path		string	true	"Module UUID"
//	@Param			file		formData	file	false	"Course material file"
//	@Param			title		formData	string	false	"Title of material"
//	@Param			description	formData	string	false	"Optional description"
//	@Param			sourceUrl	formData	string	false	"Optional iframe or external URL"
//	@Success		201			{object}	FileDto
//	@Failure		400			{object}	ginutil.ErrorResponse
//	@Failure		401			{object}	ginutil.ErrorResponse
//	@Failure		403			{object}	ginutil.ErrorResponse
//	@Failure		404			{object}	ginutil.ErrorResponse
//	@Failure		500			{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/module/{moduleUuid}/file [post]
func (ctn *ContentCtn) upload(c *gin.Context) {
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

	const maxUploadSize = 50 * 1024 * 1024 // 50MB
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize)

	title := c.PostForm("title")
	description := c.PostForm("description")
	sourceUrl := c.PostForm("sourceUrl")

	if len(title) > 255 {
		ginutil.RespondWithError(c, http.StatusBadRequest, "Title exceeds maximum length of 255 characters")
		return
	}
	if title != "" && len(title) < 2 {
		ginutil.RespondWithError(c, http.StatusBadRequest, "Title must be at least 2 characters")
		return
	}

	if sourceUrl != "" {
		parsed, err := url.ParseRequestURI(sourceUrl)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			ginutil.RespondWithError(c, http.StatusBadRequest, "Source URL must use http or https scheme")
			return
		}
	}

	var fileName string
	var fileSize int64
	var contentType string
	var fileReader io.Reader

	fileHeader, err := c.FormFile("file")
	if err == nil && fileHeader != nil {
		if fileHeader.Size > maxUploadSize {
			ginutil.RespondWithError(c, http.StatusBadRequest, "File size exceeds 50MB limit")
			return
		}

		ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
		switch ext {
		case ".html", ".htm", ".svg", ".xhtml", ".xml", ".js", ".mjs", ".exe", ".sh", ".bat", ".php", ".py", ".rb", ".cgi":
			ginutil.RespondWithError(c, http.StatusBadRequest, "File type not permitted for course materials")
			return
		}

		allowedExts := map[string]string{
			".pdf":  "application/pdf",
			".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
			".txt":  "text/plain",
			".md":   "text/plain",
			".png":  "image/png",
			".jpg":  "image/jpeg",
			".jpeg": "image/jpeg",
			".gif":  "image/gif",
			".webp": "image/webp",
			".json": "application/json",
		}

		expectedType, allowed := allowedExts[ext]
		if !allowed {
			ginutil.RespondWithError(c, http.StatusBadRequest, "Unsupported file extension")
			return
		}

		file, openErr := fileHeader.Open()
		if openErr != nil {
			ginutil.RespondWithError(c, http.StatusBadRequest, "Failed to read uploaded file: "+openErr.Error())
			return
		}
		defer file.Close()

		buffer := make([]byte, 512)
		n, _ := file.Read(buffer)
		sniffedType := http.DetectContentType(buffer[:n])
		if seeker, ok := file.(io.ReadSeeker); ok {
			_, _ = seeker.Seek(0, io.SeekStart)
		}

		if ext == ".pdf" && !strings.HasPrefix(sniffedType, "application/pdf") {
			ginutil.RespondWithError(c, http.StatusBadRequest, "File content does not match PDF format")
			return
		}

		fileName = fileHeader.Filename
		fileSize = fileHeader.Size
		contentType = expectedType
		fileReader = file
	} else if title == "" && sourceUrl == "" {
		ginutil.RespondWithError(c, http.StatusBadRequest, "File or title/sourceUrl is required")
		return
	}

	f, err := ctn.contentService.UploadFile(
		courseUuid, moduleUuid,
		title, fileName, fileSize, contentType,
		fileReader, description, sourceUrl,
	)
	if ctn.handleError(c, err) {
		return
	}

	c.JSON(http.StatusCreated, FileDto{}.FromModel(f))
}

// ListFiles godoc
//
//	@Summary		List files / materials in module
//	@Description	Returns all file materials for a given course module
//	@Tags			content
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid		path		string	true	"Course UUID"
//	@Param			moduleUuid	path		string	true	"Module UUID"
//	@Success		200			{array}		FileDto
//	@Failure		400			{object}	ginutil.ErrorResponse
//	@Failure		401			{object}	ginutil.ErrorResponse
//	@Failure		404			{object}	ginutil.ErrorResponse
//	@Failure		500			{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/module/{moduleUuid}/file [get]
func (ctn *ContentCtn) list(c *gin.Context) {
	courseUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}
	moduleUuid, ok := ginutil.ParseUUIDParam(c, "moduleUuid")
	if !ok {
		return
	}

	userUuid, _ := authutils.GetUserUUID(c)
	isPrivileged, _ := ctn.courseService.IsTeacherOrAdmin(courseUuid, userUuid, authutils.IsAdmin(c))
	if !isPrivileged {
		if !authutils.CheckCourseAccess(c, courseUuid, ctn.courseService.IsEnrolledOrAdmin, course.ErrCourseNotFound) {
			return
		}
	}

	files, err := ctn.contentService.ListFiles(courseUuid, moduleUuid, userUuid, isPrivileged)
	if ctn.handleError(c, err) {
		return
	}

	dtos := make([]FileDto, 0, len(files))
	for i := range files {
		dtos = append(dtos, FileDto{}.FromModel(&files[i]))
	}

	c.JSON(http.StatusOK, dtos)
}

// GetFile godoc
//
//	@Summary		Get file material details
//	@Description	Returns metadata for a specific file material
//	@Tags			content
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid		path		string	true	"Course UUID"
//	@Param			moduleUuid	path		string	true	"Module UUID"
//	@Param			fileUuid	path		string	true	"File content UUID"
//	@Success		200			{object}	FileDto
//	@Failure		400			{object}	ginutil.ErrorResponse
//	@Failure		401			{object}	ginutil.ErrorResponse
//	@Failure		403			{object}	ginutil.ErrorResponse
//	@Failure		404			{object}	ginutil.ErrorResponse
//	@Failure		500			{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/module/{moduleUuid}/file/{fileUuid} [get]
func (ctn *ContentCtn) get(c *gin.Context) {
	courseUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}
	moduleUuid, ok := ginutil.ParseUUIDParam(c, "moduleUuid")
	if !ok {
		return
	}
	fileUuid, ok := ginutil.ParseUUIDParam(c, "fileUuid")
	if !ok {
		return
	}

	userUuid, _ := authutils.GetUserUUID(c)
	isPrivileged, _ := ctn.courseService.IsTeacherOrAdmin(courseUuid, userUuid, authutils.IsAdmin(c))
	if !isPrivileged {
		if !authutils.CheckCourseAccess(c, courseUuid, ctn.courseService.IsEnrolledOrAdmin, course.ErrCourseNotFound) {
			return
		}
	}

	f, err := ctn.contentService.GetFile(courseUuid, moduleUuid, fileUuid, userUuid, isPrivileged)
	if ctn.handleError(c, err) {
		return
	}

	c.JSON(http.StatusOK, FileDto{}.FromModel(f))
}

// PreviewFile godoc
//
//	@Summary		Preview or stream file material
//	@Description	Streams file content with inline Content-Disposition for browser embedded preview
//	@Tags			content
//	@Security		BearerAuth
//	@Param			uuid		path	string	true	"Course UUID"
//	@Param			moduleUuid	path	string	true	"Module UUID"
//	@Param			fileUuid	path	string	true	"File content UUID"
//	@Success		200
//	@Failure		400	{object}	ginutil.ErrorResponse
//	@Failure		401	{object}	ginutil.ErrorResponse
//	@Failure		403	{object}	ginutil.ErrorResponse
//	@Failure		404	{object}	ginutil.ErrorResponse
//	@Failure		500	{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/module/{moduleUuid}/file/{fileUuid}/preview [get]
func (ctn *ContentCtn) preview(c *gin.Context) {
	courseUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}
	moduleUuid, ok := ginutil.ParseUUIDParam(c, "moduleUuid")
	if !ok {
		return
	}
	fileUuid, ok := ginutil.ParseUUIDParam(c, "fileUuid")
	if !ok {
		return
	}

	userUuid, _ := authutils.GetUserUUID(c)
	isPrivileged, _ := ctn.courseService.IsTeacherOrAdmin(courseUuid, userUuid, authutils.IsAdmin(c))
	if !isPrivileged {
		if !authutils.CheckCourseAccess(c, courseUuid, ctn.courseService.IsEnrolledOrAdmin, course.ErrCourseNotFound) {
			return
		}
	}

	reader, f, err := ctn.contentService.GetFileStream(courseUuid, moduleUuid, fileUuid, userUuid, isPrivileged)
	if ctn.handleError(c, err) {
		return
	}
	defer reader.Close()

	cleanFileName := filepath.Base(f.FileName)
	cleanFileName = strings.ReplaceAll(cleanFileName, "\r", "")
	cleanFileName = strings.ReplaceAll(cleanFileName, "\n", "")
	cleanFileName = strings.ReplaceAll(cleanFileName, "\"", "")
	disposition := mime.FormatMediaType("inline", map[string]string{"filename": cleanFileName})

	c.Header("Content-Disposition", disposition)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; sandbox")

	contentType := f.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	c.DataFromReader(http.StatusOK, f.FileSize, contentType, reader, nil)
}

// UpdateFile godoc
//
//	@Summary		Update file metadata
//	@Description	Updates title and description for course material (Teacher or Admin only)
//	@Tags			content
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid		path		string			true	"Course UUID"
//	@Param			moduleUuid	path		string			true	"Module UUID"
//	@Param			fileUuid	path		string			true	"File content UUID"
//	@Param			file		body		UpdateFileDto	true	"Updated file metadata"
//	@Success		200			{object}	FileDto
//	@Failure		400			{object}	ginutil.ErrorResponse
//	@Failure		401			{object}	ginutil.ErrorResponse
//	@Failure		403			{object}	ginutil.ErrorResponse
//	@Failure		404			{object}	ginutil.ErrorResponse
//	@Failure		500			{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/module/{moduleUuid}/file/{fileUuid} [put]
func (ctn *ContentCtn) update(c *gin.Context) {
	courseUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}
	moduleUuid, ok := ginutil.ParseUUIDParam(c, "moduleUuid")
	if !ok {
		return
	}
	fileUuid, ok := ginutil.ParseUUIDParam(c, "fileUuid")
	if !ok {
		return
	}

	if !authutils.CheckCourseAccess(c, courseUuid, ctn.courseService.IsTeacherOrAdmin, course.ErrCourseNotFound) {
		return
	}

	var dto UpdateFileDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		ginutil.RespondWithError(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	f, err := ctn.contentService.UpdateFile(courseUuid, moduleUuid, fileUuid, dto.Title, dto.Description)
	if ctn.handleError(c, err) {
		return
	}

	c.JSON(http.StatusOK, FileDto{}.FromModel(f))
}

// DeleteFile godoc
//
//	@Summary		Delete file material
//	@Description	Deletes file material from object storage and database (Teacher or Admin only)
//	@Tags			content
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid		path	string	true	"Course UUID"
//	@Param			moduleUuid	path	string	true	"Module UUID"
//	@Param			fileUuid	path	string	true	"File content UUID"
//	@Success		204
//	@Failure		400	{object}	ginutil.ErrorResponse
//	@Failure		401	{object}	ginutil.ErrorResponse
//	@Failure		403	{object}	ginutil.ErrorResponse
//	@Failure		404	{object}	ginutil.ErrorResponse
//	@Failure		500	{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/module/{moduleUuid}/file/{fileUuid} [delete]
func (ctn *ContentCtn) delete(c *gin.Context) {
	courseUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}
	moduleUuid, ok := ginutil.ParseUUIDParam(c, "moduleUuid")
	if !ok {
		return
	}
	fileUuid, ok := ginutil.ParseUUIDParam(c, "fileUuid")
	if !ok {
		return
	}

	if !authutils.CheckCourseAccess(c, courseUuid, ctn.courseService.IsTeacherOrAdmin, course.ErrCourseNotFound) {
		return
	}

	err := ctn.contentService.DeleteFile(courseUuid, moduleUuid, fileUuid)
	if ctn.handleError(c, err) {
		return
	}

	c.Status(http.StatusNoContent)
}

// CreateHandin godoc
//
//	@Summary		Create hand-in assignment in module
//	@Description	Creates a new submittable hand-in assignment (Teacher or Admin only)
//	@Tags			content
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid		path		string			true	"Course UUID"
//	@Param			moduleUuid	path		string			true	"Module UUID"
//	@Param			assignment	body		CreateHandinDto	true	"Hand-in assignment details"
//	@Success		201			{object}	HandinDto
//	@Failure		400			{object}	ginutil.ErrorResponse
//	@Failure		401			{object}	ginutil.ErrorResponse
//	@Failure		403			{object}	ginutil.ErrorResponse
//	@Failure		404			{object}	ginutil.ErrorResponse
//	@Failure		500			{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/module/{moduleUuid}/assignment [post]
func (ctn *ContentCtn) createHandin(c *gin.Context) {
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

	var dto CreateHandinDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		ginutil.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	cnt, err := ctn.contentService.CreateHandin(courseUuid, moduleUuid, dto.Title, dto.Deadline, dto.AcceptsLateSubmissions, dto.AllowsGroupSubmission)
	if ctn.handleError(c, err) {
		return
	}

	c.JSON(http.StatusCreated, HandinDto{}.FromModel(cnt))
}

// ListHandins godoc
//
//	@Summary		List hand-in assignments in module
//	@Description	Returns all hand-in assignments for a given course module
//	@Tags			content
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid		path		string	true	"Course UUID"
//	@Param			moduleUuid	path		string	true	"Module UUID"
//	@Success		200			{array}		HandinDto
//	@Failure		400			{object}	ginutil.ErrorResponse
//	@Failure		401			{object}	ginutil.ErrorResponse
//	@Failure		404			{object}	ginutil.ErrorResponse
//	@Failure		500			{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/module/{moduleUuid}/assignment [get]
func (ctn *ContentCtn) listHandins(c *gin.Context) {
	courseUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}
	moduleUuid, ok := ginutil.ParseUUIDParam(c, "moduleUuid")
	if !ok {
		return
	}

	if !authutils.CheckCourseAccess(c, courseUuid, ctn.courseService.IsEnrolledOrAdmin, course.ErrCourseNotFound) {
		return
	}

	cnts, err := ctn.contentService.ListHandins(courseUuid, moduleUuid)
	if ctn.handleError(c, err) {
		return
	}

	dtos := make([]HandinDto, 0, len(cnts))
	for i := range cnts {
		dtos = append(dtos, HandinDto{}.FromModel(&cnts[i]))
	}

	c.JSON(http.StatusOK, dtos)
}

// GetHandin godoc
//
//	@Summary		Get hand-in assignment details
//	@Description	Returns metadata for a specific hand-in assignment
//	@Tags			content
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid			path		string	true	"Course UUID"
//	@Param			moduleUuid		path		string	true	"Module UUID"
//	@Param			assignmentUuid	path		string	true	"Assignment UUID"
//	@Success		200				{object}	HandinDto
//	@Failure		400				{object}	ginutil.ErrorResponse
//	@Failure		401				{object}	ginutil.ErrorResponse
//	@Failure		404				{object}	ginutil.ErrorResponse
//	@Failure		500				{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/module/{moduleUuid}/assignment/{assignmentUuid} [get]
func (ctn *ContentCtn) getHandin(c *gin.Context) {
	courseUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}
	moduleUuid, ok := ginutil.ParseUUIDParam(c, "moduleUuid")
	if !ok {
		return
	}
	assignmentUuid, ok := ginutil.ParseUUIDParam(c, "assignmentUuid")
	if !ok {
		return
	}

	if !authutils.CheckCourseAccess(c, courseUuid, ctn.courseService.IsEnrolledOrAdmin, course.ErrCourseNotFound) {
		return
	}

	cnt, err := ctn.contentService.GetHandin(courseUuid, moduleUuid, assignmentUuid)
	if ctn.handleError(c, err) {
		return
	}

	c.JSON(http.StatusOK, HandinDto{}.FromModel(cnt))
}

// UpdateHandin godoc
//
//	@Summary		Update hand-in assignment
//	@Description	Updates title, deadline, and late submission policy (Teacher or Admin only)
//	@Tags			content
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid			path		string			true	"Course UUID"
//	@Param			moduleUuid		path		string			true	"Module UUID"
//	@Param			assignmentUuid	path		string			true	"Assignment UUID"
//	@Param			assignment		body		UpdateHandinDto	true	"Updated assignment metadata"
//	@Success		200				{object}	HandinDto
//	@Failure		400				{object}	ginutil.ErrorResponse
//	@Failure		401				{object}	ginutil.ErrorResponse
//	@Failure		403				{object}	ginutil.ErrorResponse
//	@Failure		404				{object}	ginutil.ErrorResponse
//	@Failure		409				{object}	ginutil.ErrorResponse
//	@Failure		500				{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/module/{moduleUuid}/assignment/{assignmentUuid} [put]
func (ctn *ContentCtn) updateHandin(c *gin.Context) {
	courseUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}
	moduleUuid, ok := ginutil.ParseUUIDParam(c, "moduleUuid")
	if !ok {
		return
	}
	assignmentUuid, ok := ginutil.ParseUUIDParam(c, "assignmentUuid")
	if !ok {
		return
	}

	if !authutils.CheckCourseAccess(c, courseUuid, ctn.courseService.IsTeacherOrAdmin, course.ErrCourseNotFound) {
		return
	}

	var dto UpdateHandinDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		ginutil.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	cnt, err := ctn.contentService.UpdateHandin(courseUuid, moduleUuid, assignmentUuid, dto.Title, dto.Deadline, dto.AcceptsLateSubmissions, dto.AllowsGroupSubmission)
	if ctn.handleError(c, err) {
		return
	}

	c.JSON(http.StatusOK, HandinDto{}.FromModel(cnt))
}

// DeleteHandin godoc
//
//	@Summary		Delete hand-in assignment
//	@Description	Deletes hand-in assignment and its content record (Teacher or Admin only)
//	@Tags			content
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid			path	string	true	"Course UUID"
//	@Param			moduleUuid		path	string	true	"Module UUID"
//	@Param			assignmentUuid	path	string	true	"Assignment UUID"
//	@Success		204
//	@Failure		400	{object}	ginutil.ErrorResponse
//	@Failure		401	{object}	ginutil.ErrorResponse
//	@Failure		403	{object}	ginutil.ErrorResponse
//	@Failure		404	{object}	ginutil.ErrorResponse
//	@Failure		500	{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/module/{moduleUuid}/assignment/{assignmentUuid} [delete]
func (ctn *ContentCtn) deleteHandin(c *gin.Context) {
	courseUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}
	moduleUuid, ok := ginutil.ParseUUIDParam(c, "moduleUuid")
	if !ok {
		return
	}
	assignmentUuid, ok := ginutil.ParseUUIDParam(c, "assignmentUuid")
	if !ok {
		return
	}

	if !authutils.CheckCourseAccess(c, courseUuid, ctn.courseService.IsTeacherOrAdmin, course.ErrCourseNotFound) {
		return
	}

	err := ctn.contentService.DeleteHandin(courseUuid, moduleUuid, assignmentUuid)
	if ctn.handleError(c, err) {
		return
	}

	c.Status(http.StatusNoContent)
}
