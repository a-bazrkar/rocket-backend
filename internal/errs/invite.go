package errs

import "github.com/a-bazrkar/rocket-backend/internal/packages/errcore"

func inviteDefinitions() []errcore.Definition {
	return []errcore.Definition{
		def(InviteNotFound, errcore.CategoryNotFound, "Invite not found", 404, errcore.LogInfo, false, false),
		def(InviteAlreadyExists, errcore.CategoryConflict, "Invite already exists", 409, errcore.LogInfo, false, false),
		def(InviteAccessDenied, errcore.CategoryAuthorization, "Access denied", 403, errcore.LogWarn, false, false),
	}
}
