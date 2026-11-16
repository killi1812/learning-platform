package main

import (
	"flag"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/auth"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/content"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/grade"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/info"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/notification"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/quiz"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/submission"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/seed"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/storage"
	"gorm.io/gorm"

	"go.uber.org/zap"
)

//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Type "Bearer <your-jwt-token>" to authenticate

func init() {
	flag.BoolVar(&app.RunMigrate, "migrate", false, "run database migrations")
	flag.BoolVar(&app.RunMigrate, "m", false, "run database migrations (shorthand)")
	flag.Parse()

	app.Setup()
	app.Provide(zap.S)
}

func main() {
	app.Provide(func() storage.IStorageService {
		return storage.NewSeaweedStorageService(app.SeaweedFSEndpoint)
	})
	app.Provide(user.NewUserCrudService)
	app.Provide(auth.NewAuthService)
	app.Provide(course.NewCourseService)
	app.Provide(content.NewContentService)
	app.Provide(notification.NewHub)
	app.Provide(notification.NewNotificationService)
	app.Provide(notification.NewNotifier)
	app.Provide(submission.NewSubmissionService)
	app.Provide(submission.NewSubmissionPublisher)
	app.Provide(quiz.NewQuizService)
	app.Provide(course.NewGroupService)
	app.Provide(grade.NewGradeLookup)
	app.Provide(grade.NewGradeService)

	app.RegisterApi(info.NewInfoApi)
	app.RegisterApi(user.NewUserCtn)
	app.RegisterApi(auth.NewAuthCtn)
	app.RegisterApi(course.NewCourseCtn)
	app.RegisterApi(content.NewContentCtn)
	app.RegisterApi(notification.NewNotificationCtn)
	app.RegisterApi(submission.NewSubmissionCtn)
	app.RegisterApi(quiz.NewQuizCtn)
	app.RegisterApi(grade.NewGradeCtn)
	app.RegisterWorker(submission.NewSubmissionConsumer)
	app.RegisterWorker(submission.NewDeadlineReminderWorker)

	app.RegisterModel(&user.User{})
	// Course
	app.RegisterModel(&course.Course{})
	app.RegisterModel(&course.Enrollment{})
	app.RegisterModel(&course.Group{})
	app.RegisterModel(&course.Module{})
	// Notification
	app.RegisterModel(&notification.Notification{})
	// Content
	app.RegisterModel(&content.Content{})
	app.RegisterModel(&content.File{})
	app.RegisterModel(&content.Assignment{})
	app.RegisterModel(&content.Quiz{})
	app.RegisterModel(&content.Question{})
	app.RegisterModel(&content.CodingExercise{})
	app.RegisterModel(&content.StudentInteraction{})
	// Quiz
	app.RegisterModel(&quiz.Attempt{})
	app.RegisterModel(&quiz.Answer{})
	// Submission
	app.RegisterModel(&submission.Submission{})
	app.RegisterModel(&submission.GroupSubmission{})
	app.RegisterModel(&submission.IndividualSubmission{})
	app.RegisterModel(&submission.DeadlineReminder{})
	app.RegisterModel(&submission.UploadTicket{})
	// Grade
	app.RegisterModel(&grade.Grade{})

	app.Migrate()

	app.Invoke(func(db *gorm.DB) {
		if db != nil && db.Dialector != nil && db.Dialector.Name() == "postgres" {
			_ = db.Exec("ALTER TABLE grades DROP CONSTRAINT IF EXISTS fk_submissions_grade, DROP CONSTRAINT IF EXISTS fk_group_submissions_grade").Error
		}
	})

	seed.Insert()

	app.Start()
}
