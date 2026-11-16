package definition

import (
	"github.com/a-bazrkar/rocket-backend/internal/errors/codes"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errors"
)

var (
	UserNotFound = errors.Define(
		codes.UserNotFoundErrorCode,
		codes.UserNotFoundErrorMessage,
		"The requested user could not be found.",
	)
	EmailAlreadyExists = errors.Define(
		codes.UserEmailAlreadyExistsErrorCode,
		codes.UserEmailAlreadyExistsErrorMessage,
		"An account with this email already exists.",
	)
)
