package course

import (
	"errors"
	"net/http"
	"slices"
	"strconv"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/ginutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
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

	group.POST("/join/:uuid", ctn.AddStudentToGroup)
	group.GET("/:uuid", ctn.GetGroup)                  // TODO: read a group
	group.POST("", ctn.CreateGroup)                    // TODO: create a group
	group.PATCH("/:uuid", ctn.PatchGroup)              // TODO: update a group
	group.PATCH("leave/:uuid", ctn.StudentLeavesGroup) // TODO: update a group
	group.DELETE("/:uuid", ctn.DeleteGroup)            // TODO: delete a group, note see soft delete on GORM

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
		errors.Is(err, ErrUserNotFound):
		ginutil.RespondWithError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrAlreadyEnrolled):
		ginutil.RespondWithError(c, http.StatusConflict, err.Error())
	case errors.Is(err, ErrCannotRemoveTeacher),
		errors.Is(err, ErrSelfPrerequisite),
		errors.Is(err, ErrCrossCoursePrerequisite),
		errors.Is(err, ErrCircularPrerequisite),
		errors.Is(err, ErrInvalidReorderList):
		ginutil.RespondWithError(c, http.StatusBadRequest, err.Error())
	default:
		ctn.logger.Errorf("%s: %v", fallbackMsg, err)
		ginutil.RespondWithError(c, http.StatusInternalServerError, fallbackMsg)
	}
	return true
}

func (ctn *CourseCtn) GetGroup(c *gin.Context) {
	ctn.logger.Debugf("Inside GetGroup")

	groupUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		ctn.logger.Errorf("error parsing uuid value = %s", c.Param("uuid"))
		c.Status(http.StatusInternalServerError)
		return
	}

	group, err := ctn.groupService.GetGroup(groupUuid)
	if err != nil {
		ctn.logger.Errorf("Can't delete group, %v", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	dto := NewGroupDto{}
	dto = dto.FromModel(group)
	c.AbortWithStatusJSON(http.StatusOK, dto)
}

// PostGroup godoc
//
//	@Summary	Creates a new group
//	@Tags		Groups
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Success	201		"Successfully created a group"
//	@Failure	400		{object}	ginutil.ErrorResponse	"Invalid or missing user UUID"
//	@Failure	404		{object}	ginutil.ErrorResponse	"User or group not found"
//	@Failure	500		{object}	ginutil.ErrorResponse	"Internal server error"
//	@Param		model	body		NewGroupDto				true	"Data for new group"
//	@Router		/group [post]
func (ctn *CourseCtn) CreateGroup(c *gin.Context) {
	ctn.logger.Debugf("Inside CreateGroup")

	var inputDto NewGroupDto
	if err := c.ShouldBindJSON(&inputDto); err != nil {
		ctn.logger.Errorf("Failed to bind error = %+v", err)
		ginutil.RespondWithError(c, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	newGroup, err := inputDto.ToModel()
	if err != nil {
		ginutil.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	var courseUuid uuid.UUID
	if inputDto.CourseUuid != "" {
		courseUuid, err = uuid.Parse(inputDto.CourseUuid)
		if err != nil {
			zap.S().Errorf("Failed to parse uuid = %s, parsed = %s, err = %+v", inputDto.CourseUuid, courseUuid, err)
			ginutil.RespondWithError(c, http.StatusBadRequest, "Invalid request payload: "+err.Error())
			return
		}
	}

	course, err := ctn.CourseService.GetCourse(courseUuid)
	newGroup.Course = course

	_, err = ctn.groupService.CreateGroup(newGroup)
	if err != nil {
		zap.S().Errorf("Error while creating group, err = %+v", err)
		ginutil.RespondWithError(c, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	// Log the response for debugging
	responseDto := inputDto.FromModel(newGroup)
	ctn.logger.Infof("Response DTO: %+v", responseDto)

	c.JSON(http.StatusCreated, responseDto)
}

func (ctn *CourseCtn) PatchGroup(c *gin.Context) {
	ctn.logger.Debugf("Inside PatchGroup")

	groupUuid, found := ginutil.ParseUUIDParam(c, "uuid")
	if found {
		ctn.logger.Errorf("error parsing uuid value = %s", c.Param("uuid"))
		c.Status(http.StatusInternalServerError)
		return
	}

	group, err := ctn.groupService.PatchGroup(groupUuid)
	if err != nil {
		ctn.logger.Errorf("Can't patch group, %v", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	dto := NewGroupDto{}
	dto = dto.FromModel(group)
	c.AbortWithStatusJSON(http.StatusNoContent, dto)
}

// StudentLeavesGroup godoc
//
//	@Summary	Remove student from a group
//	@Tags		Groups
//	@Accept		json
//	@Produce	json
//	@Param		uuid	path	string	true	"Group UUID"	format(uuid)
//	@Security	BearerAuth
//	@Success	200	"Successfully removed student from the group"
//	@Failure	400	{object}	ginutil.ErrorResponse	"Invalid or missing user UUID"
//	@Failure	404	{object}	ginutil.ErrorResponse	"User or group not found"
//	@Failure	500	{object}	ginutil.ErrorResponse	"Internal server error"
//	@Router		/group/leave/{uuid} [patch]
func (ctn *CourseCtn) StudentLeavesGroup(c *gin.Context) {
	ctn.logger.Debugf("Inside StudentLeavesGroup")

	var groupUuid uuid.UUID
	groupUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		ctn.logger.Errorf("error parsing uuid value = %s", c.Param("uuid"))
		c.Status(http.StatusInternalServerError)
		return
	}

	userUuid, err := authutils.GetUserUUID(c)
	if err != nil {
		ctn.logger.Errorf("Can't get user UUID, %v", err)
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	group, err := ctn.groupService.GetGroup(groupUuid)
	if err != nil {
		ctn.logger.Errorf("Can't find group, %v", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	group.Members = slices.DeleteFunc(group.Members, func(user user.User) bool {
		if user.Uuid == userUuid {
			return true
		}
		return false
	})

	ctn.groupService.RemoveStudentFromGroup(userUuid, groupUuid)
	c.AbortWithStatus(http.StatusNoContent)
}

// DeleteGroup godoc
//
//	@Summary	Deletes a group
//	@Tags		Groups
//	@Produce	json
//	@Param		uuid	path	string	true	"Group UUID"	format(uuid)
//	@Security	BearerAuth
//	@Success	204	""
//	@Failure	400	{object}	ginutil.ErrorResponse	"Invalid or missing user UUID"
//	@Failure	404	{object}	ginutil.ErrorResponse	"User or group not found"
//	@Failure	500	{object}	ginutil.ErrorResponse	"Internal server error"
//	@Router		/group/{uuid} [delete]
func (ctn *CourseCtn) DeleteGroup(c *gin.Context) {
	ctn.logger.Debugf("Inside DeleteGroup")
	groupUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		ctn.logger.Errorf("error parsing uuid value = %s", c.Param("uuid"))
		c.Status(http.StatusInternalServerError)
		return
	}

	err := ctn.groupService.DeleteGroup(groupUuid)
	if err != nil {
		ctn.logger.Errorf("Can't delete group, %v", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.AbortWithStatus(http.StatusNoContent)
}

// AddStudentToGroup godoc
//
//	@Summary		Join a group
//	@Description	Adds the authenticated student to the specified group.
//	@Tags			Groups
//	@Accept			json
//	@Produce		json
//	@Param			uuid	path	string	true	"Group UUID"	format(uuid)
//	@Security		BearerAuth
//	@Success		200	"Successfully joined the group"
//	@Failure		400	{object}	ginutil.ErrorResponse	"Invalid or missing user UUID"
//	@Failure		404	{object}	ginutil.ErrorResponse	"User or group not found"
//	@Failure		500	{object}	ginutil.ErrorResponse	"Internal server error"
//	@Router			/group/join/{uuid} [post]
func (ctn *CourseCtn) AddStudentToGroup(c *gin.Context) {
	groupUuid, ok := ginutil.ParseUUIDParam(c, "uuid")
	if !ok {
		ctn.logger.Errorf("error parsing uuid value = %s", c.Param("uuid"))
		c.Status(http.StatusInternalServerError)
		return
	}

	userUuid, err := authutils.GetUserUUID(c)
	if err != nil {
		ctn.logger.Errorf("Can't get user UUID, %v", err)
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	user, err := ctn.userService.Read(userUuid)
	if err != nil {
		ctn.logger.Errorf("Can't get user, %v", err)
		c.AbortWithError(http.StatusNotFound, err)
		return
	}

	ctn.logger.Infof("Student %v is trying to join group %v", userUuid, groupUuid)
	// TODO: service layer look at the user/db.go

	group, err := ctn.groupService.UserJoinGroup(groupUuid, user)
	ctn.logger.Debugf("Exited UserJoinGroup")

	if ginutil.HandleServiceError(c, err, gorm.ErrRecordNotFound) {
		ctn.logger.Errorf("User %v could not join group %v, %v", user, groupUuid, err)
		return
	}
	println(group)

	c.AbortWithStatus(http.StatusOK)
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
