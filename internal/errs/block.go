package errs

import "github.com/a-bazrkar/rocket-backend/internal/packages/errcore"

func blockDefinitions() []errcore.Definition {
	return []errcore.Definition{
		def(BlockNotFound, errcore.CategoryNotFound, "Block not found", 404, errcore.LogInfo, false, false),
		def(BlockAlreadyExists, errcore.CategoryConflict, "Block already exists", 409, errcore.LogInfo, false, false),
		def(BlockAccessDenied, errcore.CategoryAuthorization, "Access denied", 403, errcore.LogWarn, false, false),
	}
}
