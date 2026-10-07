// Package seed is used to create objects in database on app startup
package seed

import (
	"errors"
	"os"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"

	"go.uber.org/zap"
)

const (
	_PASSWORD_ENV = "SUPERADMIN_PASSWORD"
	_EMAIL_ENV    = "SUPERADMIN_EMAIL"
)

var suadmin *user.User

// createSuperAdmin creates a SuperAdmin user if one doesn't already exist.
func createSuperAdmin() error {
	var userCrud user.IUserCrudService
	app.Invoke(func(crud user.IUserCrudService) {
		userCrud = crud
	})

	email := os.Getenv(_EMAIL_ENV)
	if email == "" {
		email = "superadmin@example.com"
	}

	// Check if SuperAdmin exists
	{
		_, err := userCrud.GetUserByEmail(email)
		if err != nil {
			if errors.Is(err, user.ErrRecordNotFound) {
				zap.S().Infof("SuperAdmin not found, err %+v", err)
			} else {
				return err
			}
		} else {
			zap.S().Infoln("SuperAdmin found")
			zap.S().Infoln("Skipping superadmin creation")
			return nil
		}
	}

	zap.S().Infoln("Creating superadmin")
	password := os.Getenv(_PASSWORD_ENV)
	if password == "" {
		return errors.New("env variable is empty")
	}
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}

	dto := user.NewUserDto{
		Username: "superadmin",
		Email:    email,
		Password: password,
		Role:     "admin",
	}
	newUser, err := dto.ToModel()
	if err != nil {
		return err
	}

	usr, err := userCrud.Create(newUser, dto.Password)
	if err != nil {
		return err
	}
	suadmin = usr
	zap.S().Infof("superadmin created successfully, username=%s email=%s uuid=%s", usr.Username, usr.Email, usr.Uuid)
	return nil
}
