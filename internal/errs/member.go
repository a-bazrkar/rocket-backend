package errs

import "github.com/a-bazrkar/rocket-backend/internal/packages/errcore"

func memberDefinitions() []errcore.Definition {
	return []errcore.Definition{
		def(MemberNotFound, errcore.CategoryNotFound, "Member not found", 404, errcore.LogInfo, false, false),
		def(MemberAlreadyExists, errcore.CategoryConflict, "Member already exists", 409, errcore.LogInfo, false, false),
		def(MemberAccessDenied, errcore.CategoryAuthorization, "Access denied", 403, errcore.LogWarn, false, false),
	}
}
