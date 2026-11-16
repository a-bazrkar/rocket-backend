package definition

import (
	"github.com/a-bazrkar/rocket-backend/internal/errors/codes"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errors"
)

var (
	MessageNotFound = errors.Define(
		codes.MessageNotFoundErrorCode,
		codes.MessageNotFoundErrorMessage,
		"The requested message could not be found.",
	)
	MessageAlreadyExists = errors.Define(
		codes.MessageAlreadyExistsErrorCode,
		codes.MessageAlreadyExistsErrorMessage,
		"The message already exists.",
	)
)
