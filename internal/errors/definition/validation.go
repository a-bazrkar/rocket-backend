package definition

import (
	"github.com/a-bazrkar/rocket-backend/internal/errors/codes"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errors"
)

// ValidationFailed is the client-safe result of a rejected request.
var ValidationFailed = errors.Define(
	codes.ValidationFailedErrorCode,
	codes.ValidationFailedErrorMessage,
	"The request is invalid.",
)
