package errs

import "github.com/a-bazrkar/rocket-backend/internal/packages/errcore"

func userDefinitions() []errcore.Definition {
	return []errcore.Definition{
		def(UserNotFound, errcore.CategoryNotFound, "User not found", 404, errcore.LogInfo, false, false),
		def(UserAlreadyExists, errcore.CategoryConflict, "User already exists", 409, errcore.LogInfo, false, false),
		defClose(UserInvalidCredentials, errcore.CategoryAuthentication, "Invalid credentials", 401, errcore.LogWarn, false, false, errcore.CloseCodeAuth),
		def(UserAccountDisabled, errcore.CategoryAuthorization, "Account is disabled", 403, errcore.LogWarn, false, false),
	}
}
