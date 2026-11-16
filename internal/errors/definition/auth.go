package definition

import (
	"github.com/a-bazrkar/rocket-backend/internal/errors/codes"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errors"
)

var (
	Unauthorized = errors.Define(
		codes.AuthUnauthorizedErrorCode,
		codes.AuthUnauthorizedErrorMessage,
		"You are not authorized to perform this action.",
	)
	Forbidden = errors.Define(
		codes.AuthForbiddenErrorCode,
		codes.AuthForbiddenErrorMessage,
		"You do not have permission to perform this action.",
	)
)
