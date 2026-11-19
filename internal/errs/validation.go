package errs

import "github.com/a-bazrkar/rocket-backend/internal/packages/errcore"

func validationDefinitions() []errcore.Definition {
	return []errcore.Definition{
		def(ValidationInvalidInput, errcore.CategoryValidation, "Invalid input", 400, errcore.LogInfo, false, false),
		def(ValidationMissingField, errcore.CategoryValidation, "A required field is missing", 400, errcore.LogInfo, false, false),
		def(RateLimitExceeded, errcore.CategoryRateLimit, "Rate limit exceeded", 429, errcore.LogWarn, true, false),
	}
}
