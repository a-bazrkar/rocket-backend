package errs

import "github.com/a-bazrkar/rocket-backend/internal/packages/errcore"

// Definitions is the full application catalog.
// MustLoad installs it. Nothing in this package registers definitions on import.
func Definitions() []errcore.Definition {
	groups := [][]errcore.Definition{
		userDefinitions(),
		authDefinitions(),
		friendDefinitions(),
		blockDefinitions(),
		fileDefinitions(),
		conversationDefinitions(),
		memberDefinitions(),
		messageDefinitions(),
		reactionDefinitions(),
		inviteDefinitions(),
		validationDefinitions(),
		platformDefinitions(),
		internalDefinitions(),
	}
	n := 0
	for _, group := range groups {
		n += len(group)
	}
	out := make([]errcore.Definition, 0, n)
	for _, group := range groups {
		out = append(out, group...)
	}
	return out
}

// MustLoad builds the catalog and installs it as the process registry.
// A duplicate or invalid definition panics. Call it once during startup.
func MustLoad() {
	reg, err := errcore.NewRegistry(Definitions())
	if err != nil {
		panic(err)
	}
	errcore.SetDefault(reg)
}
