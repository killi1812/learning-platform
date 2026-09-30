package main

import (
	"github.com/killi1812/wfrp5e-character_sheet/app"
	"github.com/killi1812/wfrp5e-character_sheet/auth"
	"github.com/killi1812/wfrp5e-character_sheet/info"
	"github.com/killi1812/wfrp5e-character_sheet/user"
	"github.com/killi1812/wfrp5e-character_sheet/util/seed"

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
