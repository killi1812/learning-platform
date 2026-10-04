package main

import (
	"flag"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/auth"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/info"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/seed"

	"go.uber.org/zap"
	"gorm.io/gorm"
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
	app.Provide(user.NewUserCrudService)
	app.Provide(auth.NewAuthService)

	app.RegisterApi(info.NewInfoApi)
	app.RegisterApi(user.NewUserCtn)
	app.RegisterApi(auth.NewAuthCtn)

	app.Invoke(func(db *gorm.DB) {
		err := app.HandleMigrations(db, app.RunMigrate, &user.User{})
		zap.S().Errorf("Error migrating, err: %w", err)
	})

	seed.Insert()

	app.Start()
}
