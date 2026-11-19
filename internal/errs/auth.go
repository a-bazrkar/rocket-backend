package errs

import "github.com/a-bazrkar/rocket-backend/internal/packages/errcore"

func authDefinitions() []errcore.Definition {
	return []errcore.Definition{
		defClose(AuthUnauthorized, errcore.CategoryAuthentication, "Authentication required", 401, errcore.LogWarn, false, false, errcore.CloseCodeAuth),
		def(AuthForbidden, errcore.CategoryAuthorization, "You do not have permission", 403, errcore.LogWarn, false, false),
		defClose(AuthTokenExpired, errcore.CategoryAuthentication, "Token has expired", 401, errcore.LogWarn, false, false, errcore.CloseCodeAuth),
		defClose(AuthInvalidToken, errcore.CategoryAuthentication, "Invalid token", 401, errcore.LogWarn, false, false, errcore.CloseCodeAuth),
	}
}
