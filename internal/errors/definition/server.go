package definition

import (
	"github.com/a-bazrkar/rocket-backend/internal/errors/codes"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errors"
)

// ServerInternal is the generic failure returned for unexpected errors.
var ServerInternal = errors.Define(
	codes.ServerInternalErrorCode,
	codes.ServerInternalErrorMessage,
	errors.InternalDescription,
)
