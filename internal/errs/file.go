package errs

import "github.com/a-bazrkar/rocket-backend/internal/packages/errcore"

func fileDefinitions() []errcore.Definition {
	return []errcore.Definition{
		def(FileNotFound, errcore.CategoryNotFound, "File not found", 404, errcore.LogInfo, false, false),
		def(FileAlreadyExists, errcore.CategoryConflict, "File already exists", 409, errcore.LogInfo, false, false),
		def(FileAccessDenied, errcore.CategoryAuthorization, "Access denied", 403, errcore.LogWarn, false, false),
	}
}
