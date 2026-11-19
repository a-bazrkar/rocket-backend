package errs

import "github.com/a-bazrkar/rocket-backend/internal/packages/errcore"

func websocketDefinitions() []errcore.Definition {
	return []errcore.Definition{
		def(WebsocketMethodNotFound, errcore.CategoryNotFound, "Unknown method", 404, errcore.LogInfo, false, false),
		def(WebsocketInvalidMessage, errcore.CategoryValidation, "Invalid message", 400, errcore.LogInfo, false, false),
		def(WebsocketConnectionLimit, errcore.CategoryExternal, "Too many connections", 503, errcore.LogWarn, true, false),
	}
}
