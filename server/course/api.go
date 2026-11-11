package course

import (
	"errors"
	"net/http"
	"strconv"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/cerror"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/ginutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type CourseCtn struct {
	CourseService ICourseService
	logger        *zap.SugaredLogger
	userService   user.IUserCrudService
	groupService  IGroupService
}

func NewCourseCtn(courseService ICourseService, logger *zap.SugaredLogger, userService user.IUserCrudService, groupService IGroupService) app.Api {
	return &CourseCtn{
		CourseService: courseService,
		logger:        logger,
		userService:   userService,
		groupService:  groupService,
	}
}

func (ctn *CourseCtn) RegisterEndpoints(api *gin.RouterGroup) {
	courses := api.Group("/course")
	courses.Use(authutils.Protect())

	group := api.Group("/group")
	group.Use(authutils.Protect())

	// Group routes
	courses.GET("/:uuid/group", ctn.listCourseGroups)
	group.POST("", ctn.CreateGroup)
	group.POST("/bulk", ctn.BulkCreateGroups)
	group.GET("/:uuid", ctn.GetGroup)
	group.PATCH("/:uuid", ctn.PatchGroup)
	group.DELETE("/:uuid", ctn.DeleteGroup)
	group.POST("/join/:uuid", ctn.AddStudentToGroup)
	group.PATCH("/leave/:uuid", ctn.StudentLeavesGroup)

	// Course catalog routes (match both /course and /course/)
	courses.GET("", ctn.list)
	courses.GET("/", ctn.list)
	courses.GET("/:uuid", ctn.get)
	courses.GET("/:uuid/module", ctn.listModules)
	courses.GET("/:uuid/module/:moduleUuid", ctn.getModule)

	courses.GET("/my-course", ctn.getMyCourses)
	courses.POST("", ctn.create)
	courses.POST("/", ctn.create)
	courses.PUT("/:uuid", ctn.update)
	courses.DELETE("/:uuid", ctn.delete)

	// Enrollment routes
	courses.POST("/:uuid/enroll", ctn.selfEnroll)
	courses.DELETE("/:uuid/enroll", ctn.withdraw)
	courses.GET("/:uuid/enrollments", ctn.getEnrollments)
	courses.POST("/:uuid/enrollments", ctn.addEnrollment)
	courses.DELETE("/:uuid/enrollments/:userUuid", ctn.removeEnrollment)

	// Module modification routes
	courses.POST("/:uuid/module", ctn.createModule)
	courses.PUT("/:uuid/module/reorder", ctn.reorderModules)
	courses.PUT("/:uuid/module/:moduleUuid", ctn.updateModule)
	courses.DELETE("/:uuid/module/:moduleUuid", ctn.deleteModule)
}

func (ctn *CourseCtn) handleError(c *gin.Context, err error, fallbackMsg string) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, ErrCourseNotFound),
		errors.Is(err, ErrModuleNotFound),
		errors.Is(err, ErrEnrollmentNotFound),
		errors.Is(err, ErrUserNotFound),
		errors.Is(err, ErrGroupNotFound):
		ginutil.RespondWithError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrAlreadyEnrolled),
		errors.Is(err, ErrUserAlreadyInGroup),
		errors.Is(err, ErrGroupFull),
		errors.Is(err, ErrCourseGroupLimitExceeded),
		errors.Is(err, cerror.ErrGroupNameAlreadyTaken):
		ginutil.RespondWithError(c, http.StatusConflict, err.Error())
	case errors.Is(err, ErrCannotRemoveTeacher),
		errors.Is(err, ErrSelfPrerequisite),
		errors.Is(err, ErrCrossCoursePrerequisite),
		errors.Is(err, ErrCircularPrerequisite),
		errors.Is(err, ErrInvalidReorderList),
		errors.Is(err, ErrUserNotInGroup),
		errors.Is(err, ErrGroupMaxMembersTooSmall):
		ginutil.RespondWithError(c, http.StatusBadRequest, err.Error())
	default:
		ctn.logger.Errorf("%s: %v", fallbackMsg, err)
		ginutil.RespondWithError(c, http.StatusInternalServerError, fallbackMsg)
	}
	return true
}

// listCourseGroups godoc
//
//	@Summary		List groups in a course
//	@Description	Returns all groups formed for the specified course. Requires student enrollment, teacher role, or system admin.
//	@Tags			Groups
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid	path		string	true	"Course UUID"	format(uuid)
//	@Success		200		{array}		GroupDto
//	@Failure		400		{object}	ginutil.ErrorResponse	"Invalid UUID format"
//	@Failure		401		{object}	ginutil.ErrorResponse	"Unauthorized"
//	@Failure		403		{object}	ginutil.ErrorResponse	"Forbidden"
//	@Failure		404		{object}	ginutil.ErrorResponse	"Course not found"
//	@Failure		500		{object}	ginutil.ErrorResponse	"Internal server error"
//	@Router			/course/{uuid}/group [get]
func (ctn *CourseCtn) listCourseGroups(c *gin.Context) {
	courseUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}

	if !authutils.CheckCourseAccess(c, courseUuid, ctn.CourseService.IsEnrolledOrAdmin, ErrCourseNotFound) {
		return
	}

	groups, err := ctn.groupService.ListCourseGroups(courseUuid)
	if ctn.handleError(c, err, "Failed to list course groups") {
		return
	}

	dtos := make([]GroupDto, 0, len(groups))
	for i := range groups {
		dtos = append(dtos, GroupDto{}.FromModel(&groups[i]))
	}

	c.JSON(http.StatusOK, dtos)
}

// GetGroup godoc
//
//	@Summary		Get group by UUID
//	@Description	Returns details and members for the specified group. Requires course enrollment, teacher role, or system admin.
//	@Tags			Groups
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid	path		string	true	"Group UUID"	format(uuid)
//	@Success		200		{object}	GroupDto
//	@Failure		400		{object}	ginutil.ErrorResponse	"Invalid UUID format"
//	@Failure		401		{object}	ginutil.ErrorResponse	"Unauthorized"
//	@Failure		403		{object}	ginutil.ErrorResponse	"Forbidden"
//	@Failure		404		{object}	ginutil.ErrorResponse	"Group not found"
//	@Failure		500		{object}	ginutil.ErrorResponse	"Internal server error"
//	@Router			/group/{uuid} [get]
func (ctn *CourseCtn) GetGroup(c *gin.Context) {
	groupUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}

	group, err := ctn.groupService.GetGroup(groupUuid)
	if ctn.handleError(c, err, "Failed to fetch group") {
		return
	}

	if group.Course == nil {
		ginutil.RespondWithError(c, http.StatusInternalServerError, "Group course association missing")
		return
	}

	if !authutils.CheckCourseAccess(c, group.Course.Uuid, ctn.CourseService.IsEnrolledOrAdmin, ErrCourseNotFound) {
		return
	}

	c.JSON(http.StatusOK, GroupDto{}.FromModel(group))
}

// CreateGroup godoc
//
//	@Summary		Create a new group
//	@Description	Creates a new group in a course with a maximum member limit. Teacher or Admin only.
//	@Tags			Groups
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			model	body		CreateGroupDto	true	"Group creation payload"
//	@Success		201		{object}	GroupDto
//	@Failure		400		{object}	ginutil.ErrorResponse	"Invalid request payload"
//	@Failure		401		{object}	ginutil.ErrorResponse	"Unauthorized"
//	@Failure		403		{object}	ginutil.ErrorResponse	"Forbidden"
//	@Failure		404		{object}	ginutil.ErrorResponse	"Course not found"
//	@Failure		409		{object}	ginutil.ErrorResponse	"Group name already taken"
//	@Failure		500		{object}	ginutil.ErrorResponse	"Internal server error"
//	@Router			/group [post]
func (ctn *CourseCtn) CreateGroup(c *gin.Context) {
	var input CreateGroupDto
	if err := c.ShouldBindJSON(&input); err != nil {
		ginutil.RespondWithError(c, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	courseUuid, err := uuid.Parse(input.CourseUuid)
	if err != nil {
		ginutil.RespondWithError(c, http.StatusBadRequest, "Invalid course UUID")
		return
	}

	if !authutils.CheckCourseAccess(c, courseUuid, ctn.CourseService.IsTeacherOrAdmin, ErrCourseNotFound) {
		return
	}

	group, err := ctn.groupService.CreateGroup(courseUuid, input.Name, input.MaxMembers)
	if ctn.handleError(c, err, "Failed to create group") {
		return
	}

	c.JSON(http.StatusCreated, GroupDto{}.FromModel(group))
}

// BulkCreateGroups godoc
//
//	@Summary		Bulk create groups
//	@Description	Creates multiple numbered groups sequentially (e.g. Group 1 through Group 10). Teacher or Admin only.
//	@Tags			Groups
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			model	body		BulkCreateGroupsDto	true	"Bulk creation payload"
//	@Success		201		{array}		GroupDto
//	@Failure		400		{object}	ginutil.ErrorResponse	"Invalid request payload"
//	@Failure		401		{object}	ginutil.ErrorResponse	"Unauthorized"
//	@Failure		403		{object}	ginutil.ErrorResponse	"Forbidden"
//	@Failure		404		{object}	ginutil.ErrorResponse	"Course not found"
//	@Failure		500		{object}	ginutil.ErrorResponse	"Internal server error"
//	@Router			/group/bulk [post]
func (ctn *CourseCtn) BulkCreateGroups(c *gin.Context) {
	var input BulkCreateGroupsDto
	if err := c.ShouldBindJSON(&input); err != nil {
		ginutil.RespondWithError(c, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	courseUuid, err := uuid.Parse(input.CourseUuid)
	if err != nil {
		ginutil.RespondWithError(c, http.StatusBadRequest, "Invalid course UUID")
		return
	}

	if !authutils.CheckCourseAccess(c, courseUuid, ctn.CourseService.IsTeacherOrAdmin, ErrCourseNotFound) {
		return
	}

	groups, err := ctn.groupService.BulkCreateGroups(courseUuid, input.Prefix, input.Count, input.MaxMembers)
	if ctn.handleError(c, err, "Failed to bulk create groups") {
		return
	}

	dtos := make([]GroupDto, 0, len(groups))
	for i := range groups {
		dtos = append(dtos, GroupDto{}.FromModel(&groups[i]))
	}

	c.JSON(http.StatusCreated, dtos)
}

// PatchGroup godoc
//
//	@Summary		Update group
//	@Description	Modifies group name or maximum member capacity. Teacher or Admin only.
//	@Tags			Groups
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid	path		string			true	"Group UUID"	format(uuid)
//	@Param			model	body		UpdateGroupDto	true	"Group update payload"
//	@Success		200		{object}	GroupDto
//	@Failure		400		{object}	ginutil.ErrorResponse	"Invalid request payload"
//	@Failure		401		{object}	ginutil.ErrorResponse	"Unauthorized"
//	@Failure		403		{object}	ginutil.ErrorResponse	"Forbidden"
//	@Failure		404		{object}	ginutil.ErrorResponse	"Group not found"
//	@Failure		409		{object}	ginutil.ErrorResponse	"Group name already taken"
//	@Failure		500		{object}	ginutil.ErrorResponse	"Internal server error"
//	@Router			/group/{uuid} [patch]
func (ctn *CourseCtn) PatchGroup(c *gin.Context) {
	groupUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}

	group, err := ctn.groupService.GetGroup(groupUuid)
	if ctn.handleError(c, err, "Failed to fetch group") {
		return
	}

	if group.Course == nil {
		ginutil.RespondWithError(c, http.StatusInternalServerError, "Group course association missing")
		return
	}

	if !authutils.CheckCourseAccess(c, group.Course.Uuid, ctn.CourseService.IsTeacherOrAdmin, ErrCourseNotFound) {
		return
	}

	var input UpdateGroupDto
	if err := c.ShouldBindJSON(&input); err != nil {
		ginutil.RespondWithError(c, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	updated, err := ctn.groupService.UpdateGroup(groupUuid, input.Name, input.MaxMembers)
	if ctn.handleError(c, err, "Failed to update group") {
		return
	}

	c.JSON(http.StatusOK, GroupDto{}.FromModel(updated))
}

// DeleteGroup godoc
//
//	@Summary		Delete group
//	@Description	Deletes a group and clears member associations. Teacher or Admin only.
//	@Tags			Groups
//	@Produce		json
//	@Param			uuid	path	string	true	"Group UUID"	format(uuid)
//	@Security		BearerAuth
//	@Success		204	"Successfully deleted group"
//	@Failure		400	{object}	ginutil.ErrorResponse	"Invalid UUID format"
//	@Failure		401	{object}	ginutil.ErrorResponse	"Unauthorized"
//	@Failure		403	{object}	ginutil.ErrorResponse	"Forbidden"
//	@Failure		404	{object}	ginutil.ErrorResponse	"Group not found"
//	@Failure		500	{object}	ginutil.ErrorResponse	"Internal server error"
//	@Router			/group/{uuid} [delete]
func (ctn *CourseCtn) DeleteGroup(c *gin.Context) {
	groupUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}

	group, err := ctn.groupService.GetGroup(groupUuid)
	if ctn.handleError(c, err, "Failed to fetch group") {
		return
	}

	if group.Course == nil {
		ginutil.RespondWithError(c, http.StatusInternalServerError, "Group course association missing")
		return
	}

	if !authutils.CheckCourseAccess(c, group.Course.Uuid, ctn.CourseService.IsTeacherOrAdmin, ErrCourseNotFound) {
		return
	}

	err = ctn.groupService.DeleteGroup(groupUuid)
	if ctn.handleError(c, err, "Failed to delete group") {
		return
	}

	c.Status(http.StatusNoContent)
}

// AddStudentToGroup godoc
//
//	@Summary		Join a group
//	@Description	Adds the authenticated student to the specified group (enforces at most one group per course and capacity limits).
//	@Tags			Groups
//	@Accept			json
//	@Produce		json
//	@Param			uuid	path	string	true	"Group UUID"	format(uuid)
//	@Security		BearerAuth
//	@Success		200	{object}	GroupDto
//	@Failure		400	{object}	ginutil.ErrorResponse	"Invalid UUID format"
//	@Failure		401	{object}	ginutil.ErrorResponse	"Unauthorized"
//	@Failure		404	{object}	ginutil.ErrorResponse	"Group or student enrollment not found"
//	@Failure		409	{object}	ginutil.ErrorResponse	"Already in a group or group is full"
//	@Failure		500	{object}	ginutil.ErrorResponse	"Internal server error"
//	@Router			/group/join/{uuid} [post]
func (ctn *CourseCtn) AddStudentToGroup(c *gin.Context) {
	groupUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}

	userUuid, err := authutils.GetUserUUID(c)
	if err != nil {
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	group, err := ctn.groupService.GetGroup(groupUuid)
	if ctn.handleError(c, err, "Failed to fetch group") {
		return
	}

	if group.Course == nil {
		ginutil.RespondWithError(c, http.StatusInternalServerError, "Group course association missing")
		return
	}

	if !authutils.CheckCourseAccess(c, group.Course.Uuid, ctn.CourseService.IsEnrolledOrAdmin, ErrCourseNotFound) {
		return
	}

	joinedGroup, err := ctn.groupService.UserJoinGroup(groupUuid, userUuid)
	if ctn.handleError(c, err, "Failed to join group") {
		return
	}

	c.JSON(http.StatusOK, GroupDto{}.FromModel(joinedGroup))
}

// StudentLeavesGroup godoc
//
//	@Summary		Leave a group
//	@Description	Removes the authenticated student from their current group.
//	@Tags			Groups
//	@Accept			json
//	@Produce		json
//	@Param			uuid	path	string	true	"Group UUID"	format(uuid)
//	@Security		BearerAuth
//	@Success		200	{object}	GroupDto
//	@Failure		400	{object}	ginutil.ErrorResponse	"Invalid UUID format or not a group member"
//	@Failure		401	{object}	ginutil.ErrorResponse	"Unauthorized"
//	@Failure		404	{object}	ginutil.ErrorResponse	"Group or user not found"
//	@Failure		500	{object}	ginutil.ErrorResponse	"Internal server error"
//	@Router			/group/leave/{uuid} [patch]
func (ctn *CourseCtn) StudentLeavesGroup(c *gin.Context) {
	groupUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}

	userUuid, err := authutils.GetUserUUID(c)
	if err != nil {
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	group, err := ctn.groupService.GetGroup(groupUuid)
	if ctn.handleError(c, err, "Failed to fetch group") {
		return
	}

	if group.Course == nil {
		ginutil.RespondWithError(c, http.StatusInternalServerError, "Group course association missing")
		return
	}

	if !authutils.CheckCourseAccess(c, group.Course.Uuid, ctn.CourseService.IsEnrolledOrAdmin, ErrCourseNotFound) {
		return
	}

	leftGroup, err := ctn.groupService.RemoveStudentFromGroup(groupUuid, userUuid)
	if ctn.handleError(c, err, "Failed to leave group") {
		return
	}

	c.JSON(http.StatusOK, GroupDto{}.FromModel(leftGroup))
}

// ListCourses godoc
//
//	@Summary		List courses
//	@Description	Returns all courses with optional filtering by term and year
//	@Tags			course
//	@Produce		json
//	@Security		BearerAuth
//	@Param			term	query		string	false	"Term (SPRING or AUTUMN)"
//	@Param			year	query		int		false	"Academic Year (e.g. 2026)"
//	@Success		200		{array}		CourseDto
//	@Failure		401		{object}	ginutil.ErrorResponse
//	@Failure		500		{object}	ginutil.ErrorResponse
//	@Router			/course [get]
func (ctn *CourseCtn) list(c *gin.Context) {
	var termPtr *Term
	termParam := c.Query("term")
	if termParam != "" {
		t := Term(termParam)
		termPtr = &t
	}

	var yearPtr *int
	yearParam := c.Query("year")
	if yearParam != "" {
		if y, err := strconv.Atoi(yearParam); err == nil {
			yearPtr = &y
		}
	}

	courses, err := ctn.CourseService.ListCourses(termPtr, yearPtr)
	if err != nil {
		ctn.logger.Errorf("Failed to list courses: %v", err)
		ginutil.RespondWithError(c, http.StatusInternalServerError, "Failed to list courses")
		return
	}

	dtos := make([]CourseDto, 0, len(courses))
	for i := range courses {
		dtos = append(dtos, CourseDto{}.FromModel(&courses[i]))
	}
	c.JSON(http.StatusOK, dtos)
}

// GetCourse godoc
//
//	@Summary		Get course by UUID
//	@Description	Returns a single course with details
//	@Tags			course
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid	path		string	true	"Course UUID"
//	@Success		200		{object}	CourseDto
//	@Failure		400		{object}	ginutil.ErrorResponse
//	@Failure		401		{object}	ginutil.ErrorResponse
//	@Failure		404		{object}	ginutil.ErrorResponse
//	@Failure		500		{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid} [get]
func (ctn *CourseCtn) get(c *gin.Context) {
	courseUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}

	course, err := ctn.CourseService.GetCourse(courseUuid)
	if ginutil.HandleServiceError(c, err, ErrCourseNotFound) {
		return
	}

	c.JSON(http.StatusOK, CourseDto{}.FromModel(course))
}

// CreateCourse godoc
//
//	@Summary		Create course
//	@Description	Creates a new course and auto-enrolls creator as TEACHER
//	@Tags			course
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			course	body		CreateCourseDto	true	"Course information"
//	@Success		201		{object}	CourseDto
//	@Failure		400		{object}	ginutil.ErrorResponse
//	@Failure		401		{object}	ginutil.ErrorResponse
//	@Failure		500		{object}	ginutil.ErrorResponse
//	@Router			/course [post]
func (ctn *CourseCtn) create(c *gin.Context) {
	userUuid, err := authutils.GetUserUUID(c)
	if err != nil {
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var input CreateCourseDto
	if err := c.ShouldBindJSON(&input); err != nil {
		ginutil.RespondWithError(c, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	model, err := input.ToModel()
	if err != nil {
		ginutil.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	created, err := ctn.CourseService.CreateCourse(model, userUuid)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
			return
		}
		ctn.logger.Errorf("Failed to create course: %v", err)
		ginutil.RespondWithError(c, http.StatusInternalServerError, "Failed to create course")
		return
	}

	c.JSON(http.StatusCreated, CourseDto{}.FromModel(created))
}

// UpdateCourse godoc
//
//	@Summary		Update course
//	@Description	Updates course metadata (Teacher or Admin only)
//	@Tags			course
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid	path		string			true	"Course UUID"
//	@Param			course	body		UpdateCourseDto	true	"Updated course information"
//	@Success		200		{object}	CourseDto
//	@Failure		400		{object}	ginutil.ErrorResponse
//	@Failure		401		{object}	ginutil.ErrorResponse
//	@Failure		403		{object}	ginutil.ErrorResponse
//	@Failure		404		{object}	ginutil.ErrorResponse
//	@Failure		500		{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid} [put]
func (ctn *CourseCtn) update(c *gin.Context) {
	courseUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}

	if !authutils.CheckCourseAccess(c, courseUuid, ctn.CourseService.IsTeacherOrAdmin, ErrCourseNotFound) {
		return
	}

	var input UpdateCourseDto
	if err := c.ShouldBindJSON(&input); err != nil {
		ginutil.RespondWithError(c, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	model, err := input.ToModel()
	if err != nil {
		ginutil.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	updated, err := ctn.CourseService.UpdateCourse(courseUuid, model)
	if ginutil.HandleServiceError(c, err, ErrCourseNotFound) {
		return
	}

	c.JSON(http.StatusOK, CourseDto{}.FromModel(updated))
}

// DeleteCourse godoc
//
//	@Summary		Delete course
//	@Description	Deletes a course and all related modules/enrollments (Teacher or Admin only)
//	@Tags			course
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid	path	string	true	"Course UUID"
//	@Success		204
//	@Failure		400	{object}	ginutil.ErrorResponse
//	@Failure		401	{object}	ginutil.ErrorResponse
//	@Failure		403	{object}	ginutil.ErrorResponse
//	@Failure		404	{object}	ginutil.ErrorResponse
//	@Failure		500	{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid} [delete]
func (ctn *CourseCtn) delete(c *gin.Context) {
	courseUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}

	if !authutils.CheckCourseAccess(c, courseUuid, ctn.CourseService.IsTeacherOrAdmin, ErrCourseNotFound) {
		return
	}

	err := ctn.CourseService.DeleteCourse(courseUuid)
	if ginutil.HandleServiceError(c, err, ErrCourseNotFound) {
		return
	}

	c.Status(http.StatusNoContent)
}

// SelfEnroll godoc
//
//	@Summary		Self-enroll in course
//	@Description	Enrolls the authenticated user into the course as STUDENT
//	@Tags			course
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid	path		string	true	"Course UUID"
//	@Success		201		{object}	EnrollmentDto
//	@Failure		400		{object}	ginutil.ErrorResponse
//	@Failure		401		{object}	ginutil.ErrorResponse
//	@Failure		404		{object}	ginutil.ErrorResponse
//	@Failure		409		{object}	ginutil.ErrorResponse
//	@Failure		500		{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/enroll [post]
func (ctn *CourseCtn) selfEnroll(c *gin.Context) {
	courseUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}

	userUuid, err := authutils.GetUserUUID(c)
	if err != nil {
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	enrollment, err := ctn.CourseService.SelfEnroll(courseUuid, userUuid)
	if ctn.handleError(c, err, "Failed to self-enroll") {
		return
	}

	c.JSON(http.StatusCreated, EnrollmentDto{}.FromModel(enrollment))
}

// Withdraw godoc
//
//	@Summary		Withdraw from course
//	@Description	Removes the authenticated user's enrollment from the course
//	@Tags			course
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid	path	string	true	"Course UUID"
//	@Success		204
//	@Failure		400	{object}	ginutil.ErrorResponse
//	@Failure		401	{object}	ginutil.ErrorResponse
//	@Failure		404	{object}	ginutil.ErrorResponse
//	@Failure		500	{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/enroll [delete]
func (ctn *CourseCtn) withdraw(c *gin.Context) {
	courseUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}

	userUuid, err := authutils.GetUserUUID(c)
	if err != nil {
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	if ctn.handleError(c, ctn.CourseService.Withdraw(courseUuid, userUuid), "Failed to withdraw") {
		return
	}

	c.Status(http.StatusNoContent)
}

// GetMyCourses godoc
//
//	@Summary		Get current user's courses
//	@Description	Returns all courses the authenticated user is enrolled in with their roles
//	@Tags			course
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{array}		MyCourseDto
//	@Failure		401	{object}	ginutil.ErrorResponse
//	@Failure		500	{object}	ginutil.ErrorResponse
//	@Router			/course/my-course [get]
func (ctn *CourseCtn) getMyCourses(c *gin.Context) {
	userUuid, err := authutils.GetUserUUID(c)
	if err != nil {
		ginutil.RespondWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	enrollments, err := ctn.CourseService.GetUserEnrollments(userUuid)
	if err != nil {
		ctn.logger.Errorf("Failed to get user enrollments: %v", err)
		ginutil.RespondWithError(c, http.StatusInternalServerError, "Failed to get user enrollments")
		return
	}

	myCourses := make([]MyCourseDto, 0, len(enrollments))
	for _, e := range enrollments {
		if e.Course != nil {
			myCourses = append(myCourses, MyCourseDto{
				Course:     CourseDto{}.FromModel(e.Course),
				Role:       e.Role,
				EnrolledAt: e.EnrolledAt,
			})
		}
	}

	c.JSON(http.StatusOK, myCourses)
}

// GetEnrollments godoc
//
//	@Summary		Get course enrollments
//	@Description	Lists all enrollments for a course (Teacher or Admin only)
//	@Tags			course
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid	path		string	true	"Course UUID"
//	@Success		200		{array}		EnrollmentDto
//	@Failure		400		{object}	ginutil.ErrorResponse
//	@Failure		401		{object}	ginutil.ErrorResponse
//	@Failure		403		{object}	ginutil.ErrorResponse
//	@Failure		404		{object}	ginutil.ErrorResponse
//	@Failure		500		{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/enrollments [get]
func (ctn *CourseCtn) getEnrollments(c *gin.Context) {
	courseUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}

	if !authutils.CheckCourseAccess(c, courseUuid, ctn.CourseService.IsTeacherOrAdmin, ErrCourseNotFound) {
		return
	}

	enrollments, err := ctn.CourseService.GetCourseEnrollments(courseUuid)
	if ginutil.HandleServiceError(c, err, ErrCourseNotFound) {
		return
	}

	dtos := make([]EnrollmentDto, 0, len(enrollments))
	for i := range enrollments {
		dtos = append(dtos, EnrollmentDto{}.FromModel(&enrollments[i]))
	}

	c.JSON(http.StatusOK, dtos)
}

// AddEnrollment godoc
//
//	@Summary		Add user to course by email
//	@Description	Enrolls a user into the course with a specific role (Teacher or Admin only)
//	@Tags			course
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid		path		string				true	"Course UUID"
//	@Param			enrollment	body		AddEnrollmentDto	true	"Enrollment details"
//	@Success		201			{object}	EnrollmentDto
//	@Failure		400			{object}	ginutil.ErrorResponse
//	@Failure		401			{object}	ginutil.ErrorResponse
//	@Failure		403			{object}	ginutil.ErrorResponse
//	@Failure		404			{object}	ginutil.ErrorResponse
//	@Failure		409			{object}	ginutil.ErrorResponse
//	@Failure		500			{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/enrollments [post]
func (ctn *CourseCtn) addEnrollment(c *gin.Context) {
	courseUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}

	if !authutils.CheckCourseAccess(c, courseUuid, ctn.CourseService.IsTeacherOrAdmin, ErrCourseNotFound) {
		return
	}

	var input AddEnrollmentDto
	if err := c.ShouldBindJSON(&input); err != nil {
		ginutil.RespondWithError(c, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	enrollment, err := ctn.CourseService.EnrollByEmail(courseUuid, input.Email, input.Role)
	if ctn.handleError(c, err, "Failed to add enrollment") {
		return
	}

	c.JSON(http.StatusCreated, EnrollmentDto{}.FromModel(enrollment))
}

// RemoveEnrollment godoc
//
//	@Summary		Remove user from course
//	@Description	Removes a user's enrollment from the course (Teacher or Admin only)
//	@Tags			course
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid		path	string	true	"Course UUID"
//	@Param			userUuid	path	string	true	"User UUID"
//	@Success		204
//	@Failure		400	{object}	ginutil.ErrorResponse
//	@Failure		401	{object}	ginutil.ErrorResponse
//	@Failure		403	{object}	ginutil.ErrorResponse
//	@Failure		404	{object}	ginutil.ErrorResponse
//	@Failure		500	{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/enrollments/{userUuid} [delete]
func (ctn *CourseCtn) removeEnrollment(c *gin.Context) {
	courseUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}
	targetUserUuid, ok := ginutil.ParseUUIDParam(c, "userUuid")
	if !ok {
		return
	}

	if !authutils.CheckCourseAccess(c, courseUuid, ctn.CourseService.IsTeacherOrAdmin, ErrCourseNotFound) {
		return
	}

	if ctn.handleError(c, ctn.CourseService.Withdraw(courseUuid, targetUserUuid), "Failed to remove enrollment") {
		return
	}

	c.Status(http.StatusNoContent)
}

// ListModules godoc
//
//	@Summary		List modules in course
//	@Description	Returns all modules of a course ordered by position
//	@Tags			module
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid	path		string	true	"Course UUID"
//	@Success		200		{array}		ModuleDto
//	@Failure		400		{object}	ginutil.ErrorResponse
//	@Failure		401		{object}	ginutil.ErrorResponse
//	@Failure		404		{object}	ginutil.ErrorResponse
//	@Failure		500		{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/module [get]
func (ctn *CourseCtn) listModules(c *gin.Context) {
	courseUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}

	if !authutils.CheckCourseAccess(c, courseUuid, ctn.CourseService.IsEnrolledOrAdmin, ErrCourseNotFound) {
		return
	}

	modules, err := ctn.CourseService.ListModules(courseUuid)
	if ginutil.HandleServiceError(c, err, ErrCourseNotFound) {
		return
	}

	dtos := make([]ModuleDto, 0, len(modules))
	for i := range modules {
		dtos = append(dtos, ModuleDto{}.FromModel(&modules[i]))
	}

	c.JSON(http.StatusOK, dtos)
}

// GetModule godoc
//
//	@Summary		Get module
//	@Description	Returns a single module with its prerequisite dependency
//	@Tags			module
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid		path		string	true	"Course UUID"
//	@Param			moduleUuid	path		string	true	"Module UUID"
//	@Success		200			{object}	ModuleDto
//	@Failure		400			{object}	ginutil.ErrorResponse
//	@Failure		401			{object}	ginutil.ErrorResponse
//	@Failure		404			{object}	ginutil.ErrorResponse
//	@Failure		500			{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/module/{moduleUuid} [get]
func (ctn *CourseCtn) getModule(c *gin.Context) {
	courseUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}

	moduleUuid, ok := ginutil.ParseUUIDParam(c, "moduleUuid")
	if !ok {
		return
	}

	if !authutils.CheckCourseAccess(c, courseUuid, ctn.CourseService.IsEnrolledOrAdmin, ErrCourseNotFound) {
		return
	}

	module, err := ctn.CourseService.GetModule(courseUuid, moduleUuid)
	if ctn.handleError(c, err, "Failed to get module") {
		return
	}

	c.JSON(http.StatusOK, ModuleDto{}.FromModel(module))
}

// CreateModule godoc
//
//	@Summary		Create module
//	@Description	Creates a new module in a course (Teacher or Admin only)
//	@Tags			module
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid	path		string			true	"Course UUID"
//	@Param			module	body		CreateModuleDto	true	"Module information"
//	@Success		201		{object}	ModuleDto
//	@Failure		400		{object}	ginutil.ErrorResponse
//	@Failure		401		{object}	ginutil.ErrorResponse
//	@Failure		403		{object}	ginutil.ErrorResponse
//	@Failure		404		{object}	ginutil.ErrorResponse
//	@Failure		500		{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/module [post]
func (ctn *CourseCtn) createModule(c *gin.Context) {
	courseUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}

	if !authutils.CheckCourseAccess(c, courseUuid, ctn.CourseService.IsTeacherOrAdmin, ErrCourseNotFound) {
		return
	}

	var input CreateModuleDto
	if err := c.ShouldBindJSON(&input); err != nil {
		ginutil.RespondWithError(c, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	inputModel, err := input.ToModel()
	if err != nil {
		ginutil.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	created, err := ctn.CourseService.CreateModule(courseUuid, inputModel)
	if ctn.handleError(c, err, "Failed to create module") {
		return
	}

	c.JSON(http.StatusCreated, ModuleDto{}.FromModel(created))
}

// UpdateModule godoc
//
//	@Summary		Update module
//	@Description	Updates an existing module's title, position, or prerequisite (Teacher or Admin only)
//	@Tags			module
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid		path		string			true	"Course UUID"
//	@Param			moduleUuid	path		string			true	"Module UUID"
//	@Param			module		body		UpdateModuleDto	true	"Updated module information"
//	@Success		200			{object}	ModuleDto
//	@Failure		400			{object}	ginutil.ErrorResponse
//	@Failure		401			{object}	ginutil.ErrorResponse
//	@Failure		403			{object}	ginutil.ErrorResponse
//	@Failure		404			{object}	ginutil.ErrorResponse
//	@Failure		500			{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/module/{moduleUuid} [put]
func (ctn *CourseCtn) updateModule(c *gin.Context) {
	courseUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}

	moduleUuid, ok := ginutil.ParseUUIDParam(c, "moduleUuid")
	if !ok {
		return
	}

	if !authutils.CheckCourseAccess(c, courseUuid, ctn.CourseService.IsTeacherOrAdmin, ErrCourseNotFound) {
		return
	}

	var input UpdateModuleDto
	if err := c.ShouldBindJSON(&input); err != nil {
		ginutil.RespondWithError(c, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	inputModel, err := input.ToModel()
	if err != nil {
		ginutil.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	updated, err := ctn.CourseService.UpdateModule(courseUuid, moduleUuid, inputModel)
	if ctn.handleError(c, err, "Failed to update module") {
		return
	}

	c.JSON(http.StatusOK, ModuleDto{}.FromModel(updated))
}

// DeleteModule godoc
//
//	@Summary		Delete module
//	@Description	Deletes a module from the course (Teacher or Admin only)
//	@Tags			module
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid		path	string	true	"Course UUID"
//	@Param			moduleUuid	path	string	true	"Module UUID"
//	@Success		204
//	@Failure		400	{object}	ginutil.ErrorResponse
//	@Failure		401	{object}	ginutil.ErrorResponse
//	@Failure		403	{object}	ginutil.ErrorResponse
//	@Failure		404	{object}	ginutil.ErrorResponse
//	@Failure		500	{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/module/{moduleUuid} [delete]
func (ctn *CourseCtn) deleteModule(c *gin.Context) {
	courseUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}

	moduleUuid, ok := ginutil.ParseUUIDParam(c, "moduleUuid")
	if !ok {
		return
	}

	if !authutils.CheckCourseAccess(c, courseUuid, ctn.CourseService.IsTeacherOrAdmin, ErrCourseNotFound) {
		return
	}

	if ctn.handleError(c, ctn.CourseService.DeleteModule(courseUuid, moduleUuid), "Failed to delete module") {
		return
	}

	c.Status(http.StatusNoContent)
}

// ReorderModules godoc
//
//	@Summary		Reorder modules
//	@Description	Updates positions of modules sequentially according to provided UUID array (Teacher or Admin only)
//	@Tags			module
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid	path	string				true	"Course UUID"
//	@Param			body	body	ReorderModulesDto	true	"Ordered module UUIDs"
//	@Success		204
//	@Failure		400	{object}	ginutil.ErrorResponse
//	@Failure		401	{object}	ginutil.ErrorResponse
//	@Failure		403	{object}	ginutil.ErrorResponse
//	@Failure		404	{object}	ginutil.ErrorResponse
//	@Failure		500	{object}	ginutil.ErrorResponse
//	@Router			/course/{uuid}/module/reorder [put]
func (ctn *CourseCtn) reorderModules(c *gin.Context) {
	courseUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		return
	}

	if !authutils.CheckCourseAccess(c, courseUuid, ctn.CourseService.IsTeacherOrAdmin, ErrCourseNotFound) {
		return
	}

	var input ReorderModulesDto
	if err := c.ShouldBindJSON(&input); err != nil {
		ginutil.RespondWithError(c, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	if ctn.handleError(c, ctn.CourseService.ReorderModules(courseUuid, input.ModuleUuids), "Failed to reorder modules") {
		return
	}

	c.Status(http.StatusNoContent)
}
