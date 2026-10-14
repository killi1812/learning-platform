package authutils

import (
	"errors"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/cerror"
)

type Role string

const (
	ROLE_USER  Role = "user"
	ROLE_ADMIN Role = "admin"
)

func StrToRole(text string) (Role, error) {
	role := Role(text)
	switch role {
	case ROLE_ADMIN, "superadmin":
		return ROLE_ADMIN, nil
	case ROLE_USER:
		return ROLE_USER, nil

	default:
		return "", cerror.ErrUnknownRole
	}
}

var ValidUserRoles = map[Role]bool{
	ROLE_USER:  true,
	ROLE_ADMIN: true,
}

func (r Role) IsValid() bool {
	return ValidUserRoles[r]
}

func ValidateRole(role Role) error {
	if _, ok := ValidUserRoles[role]; !ok {
		return errors.New("invalid user role")
	}
	return nil
}
