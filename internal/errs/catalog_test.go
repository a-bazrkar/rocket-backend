package errs

import (
	"os"
	"testing"

	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore"
)

func TestMain(m *testing.M) {
	MustLoad()
	os.Exit(m.Run())
}

func TestCatalogMatchesRegistry(t *testing.T) {
	seen := map[errcore.Code]struct{}{}
	for _, code := range catalog {
		def, ok := errcore.Lookup(code)
		if !ok {
			t.Fatalf("missing definition for %s", code)
		}
		if def.Code != code || def.Message == "" {
			t.Fatalf("definition = %+v", def)
		}
		seen[code] = struct{}{}
	}
	for _, code := range errcore.Codes() {
		if _, ok := seen[code]; !ok {
			t.Fatalf("registered code %s has no constant", code)
		}
	}
	if len(errcore.Codes()) != len(catalog) {
		t.Fatalf("registry=%d constants=%d", len(errcore.Codes()), len(catalog))
	}
}

var catalog = []errcore.Code{
	UserNotFound,
	UserAlreadyExists,
	UserInvalidCredentials,
	UserAccountDisabled,
	AuthUnauthorized,
	AuthForbidden,
	AuthTokenExpired,
	AuthInvalidToken,
	FriendNotFound,
	FriendAlreadyExists,
	FriendAccessDenied,
	BlockNotFound,
	BlockAlreadyExists,
	BlockAccessDenied,
	FileNotFound,
	FileAlreadyExists,
	FileAccessDenied,
	ConversationNotFound,
	ConversationAlreadyExists,
	ConversationAccessDenied,
	MemberNotFound,
	MemberAlreadyExists,
	MemberAccessDenied,
	MessageNotFound,
	MessageSendFailed,
	MessageInvalidContent,
	MessageAccessDenied,
	ReactionNotFound,
	ReactionAlreadyExists,
	ReactionAccessDenied,
	InviteNotFound,
	InviteAlreadyExists,
	InviteAccessDenied,
	ValidationInvalidInput,
	ValidationMissingField,
	RateLimitExceeded,
	DatabaseQueryFailed,
	DatabaseConnectionFailed,
	ExternalServiceUnavailable,
	WebsocketMethodNotFound,
	WebsocketInvalidMessage,
	WebsocketConnectionLimit,
	InternalUnexpectedError,
	InternalPanicRecovered,
}
