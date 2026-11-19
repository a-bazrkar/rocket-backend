package errs

import "github.com/a-bazrkar/rocket-backend/internal/packages/errcore"

const (
	UserNotFound           errcore.Code = "user.not_found"
	UserAlreadyExists      errcore.Code = "user.already_exists"
	UserInvalidCredentials errcore.Code = "user.invalid_credentials"
	UserAccountDisabled    errcore.Code = "user.account_disabled"

	AuthUnauthorized errcore.Code = "auth.unauthorized"
	AuthForbidden    errcore.Code = "auth.forbidden"
	AuthTokenExpired errcore.Code = "auth.token_expired"
	AuthInvalidToken errcore.Code = "auth.invalid_token"

	FriendNotFound      errcore.Code = "friend.not_found"
	FriendAlreadyExists errcore.Code = "friend.already_exists"
	FriendAccessDenied  errcore.Code = "friend.access_denied"

	BlockNotFound      errcore.Code = "block.not_found"
	BlockAlreadyExists errcore.Code = "block.already_exists"
	BlockAccessDenied  errcore.Code = "block.access_denied"

	FileNotFound      errcore.Code = "file.not_found"
	FileAlreadyExists errcore.Code = "file.already_exists"
	FileAccessDenied  errcore.Code = "file.access_denied"

	ConversationNotFound      errcore.Code = "conversation.not_found"
	ConversationAlreadyExists errcore.Code = "conversation.already_exists"
	ConversationAccessDenied  errcore.Code = "conversation.access_denied"

	MemberNotFound      errcore.Code = "member.not_found"
	MemberAlreadyExists errcore.Code = "member.already_exists"
	MemberAccessDenied  errcore.Code = "member.access_denied"

	MessageNotFound       errcore.Code = "message.not_found"
	MessageSendFailed     errcore.Code = "message.send_failed"
	MessageInvalidContent errcore.Code = "message.invalid_content"
	MessageAccessDenied   errcore.Code = "message.access_denied"

	ReactionNotFound      errcore.Code = "reaction.not_found"
	ReactionAlreadyExists errcore.Code = "reaction.already_exists"
	ReactionAccessDenied  errcore.Code = "reaction.access_denied"

	InviteNotFound      errcore.Code = "invite.not_found"
	InviteAlreadyExists errcore.Code = "invite.already_exists"
	InviteAccessDenied  errcore.Code = "invite.access_denied"

	ValidationInvalidInput errcore.Code = "validation.invalid_input"
	ValidationMissingField errcore.Code = "validation.missing_field"
	RateLimitExceeded      errcore.Code = "rate_limit.exceeded"

	DatabaseQueryFailed        errcore.Code = "database.query_failed"
	DatabaseConnectionFailed   errcore.Code = "database.connection_failed"
	ExternalServiceUnavailable errcore.Code = "external.service_unavailable"

	WebsocketMethodNotFound  errcore.Code = "websocket.method_not_found"
	WebsocketInvalidMessage  errcore.Code = "websocket.invalid_message"
	WebsocketConnectionLimit errcore.Code = "websocket.connection_limit"

	InternalUnexpectedError errcore.Code = errcore.CodeInternalUnexpectedError
	InternalPanicRecovered  errcore.Code = errcore.CodeInternalPanicRecovered
)
