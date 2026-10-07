package auth

import (
	"errors"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/cerror"
)

type UserRole string

const (
	ROLE_USER  UserRole = "user"
	ROLE_ADMIN UserRole = "admin"
)

func StrToUserRole(text string) (UserRole, error) {
	role := UserRole(text)
	switch role {
	case ROLE_ADMIN, "superadmin":
		return ROLE_ADMIN, nil
	case ROLE_USER:
		return ROLE_USER, nil

	default:
		return "", cerror.ErrUnknownRole
	}
}

var ValidUserRoles = map[UserRole]bool{
	ROLE_USER:  true,
	ROLE_ADMIN: true,
}

var _VALID_USER_ROLES = ValidUserRoles

func (r UserRole) IsValid() bool {
	return ValidUserRoles[r]
}

func ValidateRole(role UserRole) error {
	if _, ok := _VALID_USER_ROLES[role]; !ok {
		return errors.New("invalid user role")
	}
	return nil
}
