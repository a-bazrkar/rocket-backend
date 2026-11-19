package errcore

// Category describes the nature of an error. Codes identify individual errors.
type Category string

const (
	CategoryValidation     Category = "VALIDATION"
	CategoryAuthentication Category = "AUTHENTICATION"
	CategoryAuthorization  Category = "AUTHORIZATION"
	CategoryNotFound       Category = "NOT_FOUND"
	CategoryConflict       Category = "CONFLICT"
	CategoryRateLimit      Category = "RATE_LIMIT"
	CategoryExternal       Category = "EXTERNAL"
	CategoryDatabase       Category = "DATABASE"
	CategoryInternal       Category = "INTERNAL"
)
