package cerror

import (
	"errors"
	"fmt"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/format"
)

var (
	ErrBadDateFormat      = fmt.Errorf("bad date format, should be %s", format.DateFormat)
	ErrBadDateTimeFormat  = fmt.Errorf("bad date and time format, should be %s", format.DateTimeFormat)
	ErrBadTimeFormat      = fmt.Errorf("bad time format, should be %s", format.TimeFormat)
	ErrBadUuid            = errors.New("failed to parse uuid")
	ErrUnknownRole        = errors.New("unknown role")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidTokenFormat = errors.New("invalid token format")
	ErrUserIsNil          = errors.New("user is nil")
	ErrBadRole            = errors.New("role is not allowed")
)
