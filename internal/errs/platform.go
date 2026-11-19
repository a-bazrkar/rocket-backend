package errs

import "github.com/a-bazrkar/rocket-backend/internal/packages/errcore"

func platformDefinitions() []errcore.Definition {
	return []errcore.Definition{
		def(DatabaseQueryFailed, errcore.CategoryDatabase, "A database operation failed", 500, errcore.LogError, false, false),
		def(DatabaseConnectionFailed, errcore.CategoryDatabase, "The database is unavailable", 500, errcore.LogError, true, true),
		def(ExternalServiceUnavailable, errcore.CategoryExternal, "The service is temporarily unavailable", 503, errcore.LogError, true, false),
	}
}
