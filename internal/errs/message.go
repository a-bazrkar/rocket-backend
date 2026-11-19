package errs

import "github.com/a-bazrkar/rocket-backend/internal/packages/errcore"

func messageDefinitions() []errcore.Definition {
	return []errcore.Definition{
		def(MessageNotFound, errcore.CategoryNotFound, "Message not found", 404, errcore.LogInfo, false, false),
		def(MessageSendFailed, errcore.CategoryInternal, "Unable to send message", 500, errcore.LogError, false, false),
		def(MessageInvalidContent, errcore.CategoryValidation, "Invalid message content", 400, errcore.LogInfo, false, false),
		def(MessageAccessDenied, errcore.CategoryAuthorization, "Access denied", 403, errcore.LogWarn, false, false),
	}
}
