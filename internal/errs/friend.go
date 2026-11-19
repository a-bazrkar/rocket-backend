package errs

import "github.com/a-bazrkar/rocket-backend/internal/packages/errcore"

func friendDefinitions() []errcore.Definition {
	return []errcore.Definition{
		def(FriendNotFound, errcore.CategoryNotFound, "Friend not found", 404, errcore.LogInfo, false, false),
		def(FriendAlreadyExists, errcore.CategoryConflict, "Friend already exists", 409, errcore.LogInfo, false, false),
		def(FriendAccessDenied, errcore.CategoryAuthorization, "Access denied", 403, errcore.LogWarn, false, false),
	}
}
