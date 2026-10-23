package content

import (
	"errors"
	"fmt"
	"io"
	"net/http"

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
	if logger == nil {
		logger = zap.S()
	}
	return &ContentCtn{
		contentService: contentService,
		courseService:  courseService,
		logger:         logger,
	}
}

func (ctn *ContentCtn) RegisterEndpoints(api *gin.RouterGroup) {
	pages := api.Group("/course/:uuid/module/:moduleUuid/static-page")
	pages.Use(authutils.Protect())

	pages.POST("", ctn.upload)
	pages.GET("", ctn.list)
	pages.GET("/:pageUuid", ctn.get)
	pages.GET("/:pageUuid/preview", ctn.preview)
	pages.PUT("/:pageUuid", ctn.update)
	pages.DELETE("/:pageUuid", ctn.delete)
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
	default:
		ctn.logger.Errorf("Unexpected content error: %v", err)
		ginutil.RespondWithError(c, http.StatusInternalServerError, "Internal server error")
		return true
	}
}

// UploadStaticPage godoc
//
//	@Summary		Upload or create static page material
//	@Description	Uploads course material file (.pdf, .pptx, .md, .txt) to object storage or creates link (Teacher or Admin only)
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
//	@Success		201			{object}	StaticPageDto
//	@Failure		400			{object}	ginutil.ErrorResponse
//	@Failure		401			{object}	ginutil.ErrorResponse
//	@Failure		403			{object}	ginutil.ErrorResponse
//	@Failure		404			{object}	ginutil.ErrorResponse
//	@Failure		500			{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/module/{moduleUuid}/static-page [post]
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

	title := c.PostForm("title")
	description := c.PostForm("description")
	sourceUrl := c.PostForm("sourceUrl")

	var fileName string
	var fileSize int64
	var contentType string
	var fileReader io.Reader

	fileHeader, err := c.FormFile("file")
	if err == nil && fileHeader != nil {
		file, openErr := fileHeader.Open()
		if openErr != nil {
			ginutil.RespondWithError(c, http.StatusBadRequest, "Failed to read uploaded file: "+openErr.Error())
			return
		}
		defer file.Close()

		fileName = fileHeader.Filename
		fileSize = fileHeader.Size
		contentType = fileHeader.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		fileReader = file
	} else if title == "" && sourceUrl == "" {
		ginutil.RespondWithError(c, http.StatusBadRequest, "File or title/sourceUrl is required")
		return
	}

	sp, err := ctn.contentService.UploadStaticPage(
		courseUuid, moduleUuid,
		title, fileName, fileSize, contentType,
		fileReader, description, sourceUrl,
	)
	if ctn.handleError(c, err) {
		return
	}

	c.JSON(http.StatusCreated, StaticPageDto{}.FromModel(sp))
}

// ListStaticPages godoc
//
//	@Summary		List static pages / materials in module
//	@Description	Returns all static page materials for a given course module
//	@Tags			content
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid		path		string	true	"Course UUID"
//	@Param			moduleUuid	path		string	true	"Module UUID"
//	@Success		200			{array}		StaticPageDto
//	@Failure		400			{object}	ginutil.ErrorResponse
//	@Failure		401			{object}	ginutil.ErrorResponse
//	@Failure		404			{object}	ginutil.ErrorResponse
//	@Failure		500			{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/module/{moduleUuid}/static-page [get]
func (ctn *ContentCtn) list(c *gin.Context) {
	courseUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}
	moduleUuid, ok := ginutil.ParseUUIDParam(c, "moduleUuid")
	if !ok {
		return
	}

	pages, err := ctn.contentService.ListStaticPages(courseUuid, moduleUuid)
	if ctn.handleError(c, err) {
		return
	}

	dtos := make([]StaticPageDto, 0, len(pages))
	for i := range pages {
		dtos = append(dtos, StaticPageDto{}.FromModel(&pages[i]))
	}

	c.JSON(http.StatusOK, dtos)
}

// GetStaticPage godoc
//
//	@Summary		Get static page material details
//	@Description	Returns metadata for a specific static page
//	@Tags			content
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid		path		string	true	"Course UUID"
//	@Param			moduleUuid	path		string	true	"Module UUID"
//	@Param			pageUuid	path		string	true	"Static page content UUID"
//	@Success		200			{object}	StaticPageDto
//	@Failure		400			{object}	ginutil.ErrorResponse
//	@Failure		401			{object}	ginutil.ErrorResponse
//	@Failure		404			{object}	ginutil.ErrorResponse
//	@Failure		500			{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/module/{moduleUuid}/static-page/{pageUuid} [get]
func (ctn *ContentCtn) get(c *gin.Context) {
	courseUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}
	moduleUuid, ok := ginutil.ParseUUIDParam(c, "moduleUuid")
	if !ok {
		return
	}
	pageUuid, ok := ginutil.ParseUUIDParam(c, "pageUuid")
	if !ok {
		return
	}

	sp, err := ctn.contentService.GetStaticPage(courseUuid, moduleUuid, pageUuid)
	if ctn.handleError(c, err) {
		return
	}

	c.JSON(http.StatusOK, StaticPageDto{}.FromModel(sp))
}

// PreviewStaticPage godoc
//
//	@Summary		Preview or stream static page material file
//	@Description	Streams file content with inline Content-Disposition for browser embedded preview
//	@Tags			content
//	@Security		BearerAuth
//	@Param			uuid		path	string	true	"Course UUID"
//	@Param			moduleUuid	path	string	true	"Module UUID"
//	@Param			pageUuid	path	string	true	"Static page content UUID"
//	@Success		200
//	@Failure		400	{object}	ginutil.ErrorResponse
//	@Failure		401	{object}	ginutil.ErrorResponse
//	@Failure		404	{object}	ginutil.ErrorResponse
//	@Failure		500	{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/module/{moduleUuid}/static-page/{pageUuid}/preview [get]
func (ctn *ContentCtn) preview(c *gin.Context) {
	courseUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}
	moduleUuid, ok := ginutil.ParseUUIDParam(c, "moduleUuid")
	if !ok {
		return
	}
	pageUuid, ok := ginutil.ParseUUIDParam(c, "pageUuid")
	if !ok {
		return
	}

	reader, sp, err := ctn.contentService.GetFileStream(courseUuid, moduleUuid, pageUuid)
	if ctn.handleError(c, err) {
		return
	}
	defer reader.Close()

	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", sp.FileName))
	contentType := sp.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	c.DataFromReader(http.StatusOK, sp.FileSize, contentType, reader, nil)
}

// UpdateStaticPage godoc
//
//	@Summary		Update static page metadata
//	@Description	Updates title and description for course material (Teacher or Admin only)
//	@Tags			content
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid		path		string				true	"Course UUID"
//	@Param			moduleUuid	path		string				true	"Module UUID"
//	@Param			pageUuid	path		string				true	"Static page content UUID"
//	@Param			page		body		UpdateStaticPageDto	true	"Updated page metadata"
//	@Success		200			{object}	StaticPageDto
//	@Failure		400			{object}	ginutil.ErrorResponse
//	@Failure		401			{object}	ginutil.ErrorResponse
//	@Failure		403			{object}	ginutil.ErrorResponse
//	@Failure		404			{object}	ginutil.ErrorResponse
//	@Failure		500			{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/module/{moduleUuid}/static-page/{pageUuid} [put]
func (ctn *ContentCtn) update(c *gin.Context) {
	courseUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}
	moduleUuid, ok := ginutil.ParseUUIDParam(c, "moduleUuid")
	if !ok {
		return
	}
	pageUuid, ok := ginutil.ParseUUIDParam(c, "pageUuid")
	if !ok {
		return
	}

	if !authutils.CheckCourseAccess(c, courseUuid, ctn.courseService.IsTeacherOrAdmin, course.ErrCourseNotFound) {
		return
	}

	var dto UpdateStaticPageDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		ginutil.RespondWithError(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	sp, err := ctn.contentService.UpdateStaticPage(courseUuid, moduleUuid, pageUuid, dto.Title, dto.Description)
	if ctn.handleError(c, err) {
		return
	}

	c.JSON(http.StatusOK, StaticPageDto{}.FromModel(sp))
}

// DeleteStaticPage godoc
//
//	@Summary		Delete static page material
//	@Description	Deletes static page material from object storage and database (Teacher or Admin only)
//	@Tags			content
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid		path	string	true	"Course UUID"
//	@Param			moduleUuid	path	string	true	"Module UUID"
//	@Param			pageUuid	path	string	true	"Static page content UUID"
//	@Success		204
//	@Failure		400	{object}	ginutil.ErrorResponse
//	@Failure		401	{object}	ginutil.ErrorResponse
//	@Failure		403	{object}	ginutil.ErrorResponse
//	@Failure		404	{object}	ginutil.ErrorResponse
//	@Failure		500	{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/module/{moduleUuid}/static-page/{pageUuid} [delete]
func (ctn *ContentCtn) delete(c *gin.Context) {
	courseUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}
	moduleUuid, ok := ginutil.ParseUUIDParam(c, "moduleUuid")
	if !ok {
		return
	}
	pageUuid, ok := ginutil.ParseUUIDParam(c, "pageUuid")
	if !ok {
		return
	}

	if !authutils.CheckCourseAccess(c, courseUuid, ctn.courseService.IsTeacherOrAdmin, course.ErrCourseNotFound) {
		return
	}

	err := ctn.contentService.DeleteStaticPage(courseUuid, moduleUuid, pageUuid)
	if ctn.handleError(c, err) {
		return
	}

	c.Status(http.StatusNoContent)
}
