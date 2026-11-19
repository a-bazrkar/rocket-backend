package errs

import "github.com/a-bazrkar/rocket-backend/internal/packages/errcore"

func reactionDefinitions() []errcore.Definition {
	return []errcore.Definition{
		def(ReactionNotFound, errcore.CategoryNotFound, "Reaction not found", 404, errcore.LogInfo, false, false),
		def(ReactionAlreadyExists, errcore.CategoryConflict, "Reaction already exists", 409, errcore.LogInfo, false, false),
		def(ReactionAccessDenied, errcore.CategoryAuthorization, "Access denied", 403, errcore.LogWarn, false, false),
	}
}
