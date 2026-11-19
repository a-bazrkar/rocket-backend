package errs

import "github.com/a-bazrkar/rocket-backend/internal/packages/errcore"

func conversationDefinitions() []errcore.Definition {
	return []errcore.Definition{
		def(ConversationNotFound, errcore.CategoryNotFound, "Conversation not found", 404, errcore.LogInfo, false, false),
		def(ConversationAlreadyExists, errcore.CategoryConflict, "Conversation already exists", 409, errcore.LogInfo, false, false),
		def(ConversationAccessDenied, errcore.CategoryAuthorization, "Access denied", 403, errcore.LogWarn, false, false),
	}
}
