package errs

import "github.com/a-bazrkar/rocket-backend/internal/packages/errcore"

func internalDefinitions() []errcore.Definition {
	return []errcore.Definition{
		def(InternalUnexpectedError, errcore.CategoryInternal, "An unexpected error occurred", 500, errcore.LogError, false, true),
		def(InternalPanicRecovered, errcore.CategoryInternal, "An unexpected error occurred", 500, errcore.LogError, false, true),
	}
}
