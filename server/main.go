package main

import (
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/auth"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/info"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/seed"

	"go.uber.org/zap"
)

//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Type "Bearer <your-jwt-token>" to authenticate

func init() {
	app.Setup()
	app.Provide(zap.S)
}

func main() {
	app.Provide(user.NewUserCrudService)
	app.Provide(auth.NewAuthService)

	app.RegisterApi(info.NewInfoApi)
	app.RegisterApi(user.NewUserCtn)
	app.RegisterApi(auth.NewAuthCtn)

	seed.Insert()

	app.Start()
}
