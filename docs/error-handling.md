# Error handling

Specification and implementation notes for the Rocket backend error system.

Services return ordinary Go errors. One registry decides the public code, the public message, the HTTP status, and the log severity. HTTP and WebSocket only render that decision. They do not invent a second error model.

## Boundaries

| Path | Role |
|---|---|
| `internal/errs` | Application catalog. This is the only place to add a product error. |
| `internal/packages/errcore` | Engine: types, registry, constructors, request context. No Gin, Gorilla, or Zerolog. |
| `internal/packages/errcore/classify` | `Classify` turns any `error` into a classification and does not log. `Log` writes one event through `logger.Logger`. |
| `internal/packages/errcore/transport` | HTTP JSON and WebSocket frames. |
| `internal/packages/errcore/recovery` | Panic recovery for Gin handlers and WebSocket goroutines. |

Import rules:

- A service imports `internal/errs` for codes and `internal/packages/errcore` for `New` and `Wrap`. It does not write an HTTP body or a WebSocket frame.
- A handler imports `internal/packages/errcore/transport` and calls `transport.AbortWithError`.
- Nothing in `internal/errs` imports Gin, Gorilla, or Zerolog.
- `internal/packages/errcore` does not import `internal/errs`. The catalog does not register itself on import.
- `rest.New` calls `errs.MustLoad`, which builds one immutable registry with `NewRegistry` and installs it with `SetDefault`.
- The server calls `errcore.SetDebug(cfg.Debug)` before it accepts traffic. `APP_DEBUG` defaults to false.

```text
service or repository
        |
        |  errcore.Wrap(ctx, errs.UserNotFound, err)
        v
internal/errs  --MustLoad-->  packages/errcore registry
        |
        |  classify.Classify
        v
classification
        |
        +--> application logger (once)
        +--> HTTP JSON  or  WebSocket event / close frame
```

## Codes

A code is a stable public identifier.

```go
type Code string
```

Format, enforced by the registry:

```text
^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$
```

Examples: `user.not_found`, `rate_limit.exceeded`, `database.query_failed`.

Rules:

1. Every code is unique.
2. A code keeps its meaning after a client has seen it. Changing the meaning requires a new code.
3. Clients branch on `code`. They do not branch on `message`.
4. Messages are for people. They are not identifiers.
5. There are no numeric error IDs and no service-name prefixes.

`internal.unexpected_error` and `internal.panic_recovered` are declared on the engine because recovery and the unknown-error fallback refer to those strings. Their public messages are still registered from `internal/errs/internal.go`.

## Categories

A category describes the kind of failure. The code identifies which failure.

```go
type Category string

const (
    CategoryValidation     Category = "VALIDATION"
    CategoryAuthentication Category = "AUTHENTICATION"
    CategoryAuthorization  Category = "AUTHORIZATION"
    CategoryNotFound       Category = "NOT_FOUND"
    CategoryConflict       Category = "CONFLICT"
    CategoryRateLimit      Category = "RATE_LIMIT"
    CategoryExternal       Category = "EXTERNAL"
    CategoryDatabase       Category = "DATABASE"
    CategoryInternal       Category = "INTERNAL"
)
```

A category does not decide whether a WebSocket connection closes. That is the optional socket policy on the definition. See [WebSocket](#websocket).

## Definitions

```go
type Definition struct {
    Code        Code
    Category    Category
    Message     string // client-facing
    HTTPStatus  int    // 400–599, plain int, no net/http import
    Retryable   bool
    LogLevel    LogLevel // debug, info, warn, error
    ShouldAlert bool
    Socket      SocketPolicy // empty, close, or event_close
    CloseCode   int          // 4000–4999 when Socket closes
}
```

`Message` is the only string a client may see for that code. `HTTPStatus` is stored as an integer so the engine does not depend on `net/http`.

`Socket` is empty unless that error should close a WebSocket. `close` sends only a close frame. `event_close` sends the error event and then the close frame. `CloseCode` is a private-use code in 4000–4999. The engine does not import Gorilla.

`NewRegistry` returns an error when any of these are true:

- the code fails the format above
- the category is not one of the constants
- the message is empty
- the HTTP status is outside 400–599
- the log level is not `debug`, `info`, `warn`, or `error`
- the code is duplicated inside the slice
- `Socket` is not empty, `close`, or `event_close`
- a close policy has no close code in 4000–4999
- a close code is set on the default (open) policy

`MustLoad` panics when `NewRegistry` returns an error. The registry map is not modified after `NewRegistry` returns. `SetDefault` publishes it. `Lookup` and `Codes` read that pointer. `Validate` checks a slice without writing a registry.

Unknown errors use `Fallback()`:

| Field | Value |
|---|---|
| Code | `internal.unexpected_error` |
| Category | `INTERNAL` |
| Message | `An unexpected error occurred` |
| HTTP status | 500 |
| Log level | error |
| ShouldAlert | true |

If that code has been registered, `Fallback` uses the registered definition. The hardcoded row above is what the engine returns before registration.

## Application error

```go
type Error struct {
    Code     Code
    Category Category
    Message  string // registry message copied at construction
    Cause    error
    Metadata map[string]any
    detail   string // unexported, logs only
}
```

Behavior:

- `Error()` is an internal description. It includes the code, the registry message, the internal detail, and `Cause.Error()`. Logs may use it. Clients must not.
- `Unwrap()` returns `Cause`, so `errors.Is` and `errors.As` walk the chain.
- `Is` matches another `*Error` with the same non-empty code. `errors.Is(err, errcore.New(ctx, errs.UserNotFound))` is true for any wrap of that code.
- `WithMetadata` returns a copy. The receiver is unchanged. Empty keys are dropped.
- Construction copies request, trace, user, conversation, and connection ids from `ctx`. A nil context records nothing. The `context.Context` value is not stored. `classify.Log` prefers those ids and fills gaps from the context passed at log time.

Constructors live on `errcore`. Codes live on `errs`.

```go
func New(ctx context.Context, code Code) *Error
func Wrap(ctx context.Context, code Code, cause error) *Error
func WrapWithMessage(ctx context.Context, code Code, cause error, detail string) *Error
```

`New` and `Wrap` copy the category and the public message from the registry. An unknown code still produces an `*Error` with category `INTERNAL` and an empty message. `Classify` then hides it behind `internal.unexpected_error`. When `errcore.Debug` is on, they also record a short caller stack.

`WrapWithMessage` stores `detail` for logs. It does not change the client message. Passing `err.Error()` here cannot leak that string to a client.

There is no package-level mutable sentinel. Each call returns a new value.

## Catalog

Severity defaults used by the current definitions:

| Situation | HTTP | Log level |
|---|---|---|
| Validation | 400 | info |
| Not found | 404 | info |
| Conflict | 409 | info |
| Invalid credentials, unauthorized, expired or invalid token | 401 | warn |
| Forbidden, access denied, disabled account | 403 | warn |
| Rate limit | 429 | warn |
| Database failure, `message.send_failed` | 500 | error |
| External service unavailable | 503 | error |
| Unexpected error, recovered panic | 500 | error |

`Retryable` is set on `rate_limit.exceeded`, `database.connection_failed`, and `external.service_unavailable`.

`ShouldAlert` is set on `database.connection_failed`, `internal.unexpected_error`, and `internal.panic_recovered`. Nothing sends an alert yet. The flag is stored so a later hook can.

| Constant | Code | Category | Message |
|---|---|---|---|
| `UserNotFound` | `user.not_found` | `NOT_FOUND` | User not found |
| `UserAlreadyExists` | `user.already_exists` | `CONFLICT` | User already exists |
| `UserInvalidCredentials` | `user.invalid_credentials` | `AUTHENTICATION` | Invalid credentials |
| `UserAccountDisabled` | `user.account_disabled` | `AUTHORIZATION` | Account is disabled |
| `AuthUnauthorized` | `auth.unauthorized` | `AUTHENTICATION` | Authentication required |
| `AuthForbidden` | `auth.forbidden` | `AUTHORIZATION` | You do not have permission |
| `AuthTokenExpired` | `auth.token_expired` | `AUTHENTICATION` | Token has expired |
| `AuthInvalidToken` | `auth.invalid_token` | `AUTHENTICATION` | Invalid token |
| `FriendNotFound` | `friend.not_found` | `NOT_FOUND` | Friend not found |
| `FriendAlreadyExists` | `friend.already_exists` | `CONFLICT` | Friend already exists |
| `FriendAccessDenied` | `friend.access_denied` | `AUTHORIZATION` | Access denied |
| `BlockNotFound` | `block.not_found` | `NOT_FOUND` | Block not found |
| `BlockAlreadyExists` | `block.already_exists` | `CONFLICT` | Block already exists |
| `BlockAccessDenied` | `block.access_denied` | `AUTHORIZATION` | Access denied |
| `FileNotFound` | `file.not_found` | `NOT_FOUND` | File not found |
| `FileAlreadyExists` | `file.already_exists` | `CONFLICT` | File already exists |
| `FileAccessDenied` | `file.access_denied` | `AUTHORIZATION` | Access denied |
| `ConversationNotFound` | `conversation.not_found` | `NOT_FOUND` | Conversation not found |
| `ConversationAlreadyExists` | `conversation.already_exists` | `CONFLICT` | Conversation already exists |
| `ConversationAccessDenied` | `conversation.access_denied` | `AUTHORIZATION` | Access denied |
| `MemberNotFound` | `member.not_found` | `NOT_FOUND` | Member not found |
| `MemberAlreadyExists` | `member.already_exists` | `CONFLICT` | Member already exists |
| `MemberAccessDenied` | `member.access_denied` | `AUTHORIZATION` | Access denied |
| `MessageNotFound` | `message.not_found` | `NOT_FOUND` | Message not found |
| `MessageSendFailed` | `message.send_failed` | `INTERNAL` | Unable to send message |
| `MessageInvalidContent` | `message.invalid_content` | `VALIDATION` | Invalid message content |
| `MessageAccessDenied` | `message.access_denied` | `AUTHORIZATION` | Access denied |
| `ReactionNotFound` | `reaction.not_found` | `NOT_FOUND` | Reaction not found |
| `ReactionAlreadyExists` | `reaction.already_exists` | `CONFLICT` | Reaction already exists |
| `ReactionAccessDenied` | `reaction.access_denied` | `AUTHORIZATION` | Access denied |
| `InviteNotFound` | `invite.not_found` | `NOT_FOUND` | Invite not found |
| `InviteAlreadyExists` | `invite.already_exists` | `CONFLICT` | Invite already exists |
| `InviteAccessDenied` | `invite.access_denied` | `AUTHORIZATION` | Access denied |
| `ValidationInvalidInput` | `validation.invalid_input` | `VALIDATION` | Invalid input |
| `ValidationMissingField` | `validation.missing_field` | `VALIDATION` | A required field is missing |
| `RateLimitExceeded` | `rate_limit.exceeded` | `RATE_LIMIT` | Rate limit exceeded |
| `DatabaseQueryFailed` | `database.query_failed` | `DATABASE` | A database operation failed |
| `DatabaseConnectionFailed` | `database.connection_failed` | `DATABASE` | The database is unavailable |
| `ExternalServiceUnavailable` | `external.service_unavailable` | `EXTERNAL` | The service is temporarily unavailable |
| `InternalUnexpectedError` | `internal.unexpected_error` | `INTERNAL` | An unexpected error occurred |
| `InternalPanicRecovered` | `internal.panic_recovered` | `INTERNAL` | An unexpected error occurred |

Login failure is `user.invalid_credentials`. There is no `auth.invalid_credentials`.

Domain files: `user.go`, `auth.go`, `friend.go`, `block.go`, `file.go`, `conversation.go`, `member.go`, `message.go`, `reaction.go`, `invite.go`, `validation.go`, `platform.go`, `internal.go`. Codes live in `code.go`. Shared helpers live in `def.go`.

## Classification

```go
func Classify(err error) Classification
```

`Classification` carries the public code, category, message, and HTTP status, plus log level, `ShouldLog`, `ShouldAlert`, `Retryable`, the original `Cause`, redacted metadata, and an optional stack string.

Rules, in order:

1. `nil` returns the zero value. `ShouldLog` is false. Callers must not write a response for it.
2. `errors.As` finds a `*errors.Error` anywhere in the chain.
3. A registered code uses that definition's message, status, level, retry flag, and alert flag. The public message is the registry message, never `err.Error()` and never the unexported detail.
4. A plain error, or an `*errors.Error` whose code was never registered, becomes `internal.unexpected_error`. The original error stays on `Cause` for logs. An unregistered code is also stored as metadata `unregistered_code`.
5. Metadata is copied and redacted before it is placed on the classification.
6. `Stack` is empty unless recovery sets it. It is log-only.

`MetricKey(transport)` returns `code`, `category`, and `transport`. Those are the only fields safe to use as Prometheus labels later. User IDs, request IDs, conversation IDs, connection IDs, and raw messages are excluded. Prometheus is not a dependency.

## Logging

`classify.Log` writes one structured event through `logger.Logger` when `ShouldLog` is true and the logger level allows the event. Zerolog is the current implementation of that interface and is imported only by `internal/packages/logger`.

Fields, when present:

| Field | Source |
|---|---|
| `error_code` | classification code |
| `error_category` | classification category |
| `error` | original cause, via the logger `Err` field |
| `request_id` | context |
| `trace_id` | context |
| `user_id` | context |
| `conversation_id` | context |
| `connection_id` | context |
| `component` | caller, such as `http`, `gin`, or `websocket` |
| `transport` | `http` or `websocket` |
| `metadata` | one object, after redaction |
| `stack` | recovered panic only |
| `alert` | `ShouldAlert` |
| `retryable` | definition flag |

The event message is `error`.

Context keys live on the engine (`WithRequestID`, `RequestID`, and the same pair for trace ID, user ID, conversation ID, and connection ID). `New` and `Wrap` copy those ids onto the error. `classify.Log` prefers that copy and fills any missing id from the context passed at log time. Empty strings are not stored. Integer IDs are stored even when zero if `WithUserID` or `WithConversationID` is called. WebSocket panic recovery treats user ID `0` and conversation ID `0` as absent and does not call those setters.

Redaction applies to metadata keys, case-insensitive, with `-` treated as `_`. Exact keys: `password`, `password_hash`, `passwd`, `secret`, `session`, `token`, `access_token`, `refresh_token`, `id_token`, `authorization`, `cookie`, `set_cookie`, `content`, `message`, `body`. Any key containing `password`, `secret`, or `token` is redacted. The replacement value is `[redacted]`. Nested maps and slices are walked. Metadata is one log object. Its keys are never promoted to logger field names. Redaction is a backstop. Callers still must not put secrets in metadata. Client responses do not include metadata.

Who logs:

- `transport.AbortWithError` logs once, then writes.
- `middleware.Errors` logs only a leftover `c.Errors` entry, by calling `transport.AbortWithError`, when the body has not been written and `AbortWithError` has not already handled the request.
- `transport.WriteHTTPError` does not log. Panic recovery logs first, then calls it.
- A second `AbortWithError` on the same request does not log again and does not replace a written body.

Validation and not-found stay at info. Authentication and authorization stay at warn. Database, external, and internal failures are error. A disabled logger level drops the event.

## Public responses

Clients receive only `code` and `message` from a registered definition.

They never receive:

- stack traces, except when `APP_DEBUG=true`, which adds a `stack` array for local debugging
- `Cause` or `err.Error()`
- SQL, driver text, or connection strings
- file paths
- passwords, tokens, or message bodies
- the panic value

### HTTP

```json
{
  "error": {
    "code": "user.not_found",
    "message": "User not found"
  }
}
```

Status comes from the classification. Content type is JSON. The Gin context is aborted.

With `APP_DEBUG=true` the same object also contains `stack`. With `APP_DEBUG=false` that key is absent. The cause text is absent in both modes. WebSocket error events use the same object.

`WriteHTTPError` leaves a response alone when `c.Writer.Written()` is already true, and still aborts. An empty code or a status outside 400–599 is replaced with the fallback before the body is written.

Unknown error body:

```json
{
  "error": {
    "code": "internal.unexpected_error",
    "message": "An unexpected error occurred"
  }
}
```

Status 500.

Handler usage once routes exist:

```go
if err != nil {
    transport.AbortWithError(c, err)
    return
}
```

`c.Error(err)` without writing a body is also handled. The middleware uses the last attached error. Prefer `AbortWithError` and return, so the handler stops.

### WebSocket

Recoverable errors are one text frame. The connection stays open.

```json
{
  "type": "error",
  "error": {
    "code": "message.send_failed",
    "message": "Unable to send message"
  }
}
```

`Respond` reads the socket policy on the classification. The default sends the error event and leaves the connection open. `auth.forbidden` and other authorization errors use that default. `SocketClose` sends a close frame and no event. `user.invalid_credentials`, `auth.unauthorized`, `auth.token_expired`, and `auth.invalid_token` use `SocketClose` with code `4001`. `SocketEventClose` sends the event and then the close frame. The close reason is the public message, cut to 123 bytes on a UTF-8 boundary.

`Respond` runs on the caller's goroutine. That caller must already be the only data-frame writer. This package does not take a write mutex.

`WriteConn` and `WriteEvent` use `WriteJSON`. The caller must be the only goroutine writing data frames. A write error is returned. The function does not write a second frame.

`CloseConn` and `WriteClose` use `WriteControl`. Gorilla allows that beside another data writer. Application close codes are in the private range 4000–4999. `WriteClose` rejects any other code.

| Code | Meaning |
|---|---|
| 4000 | Recovered panic |
| 4001 | Authentication failure |

There is no WebSocket server in this repository yet. The write pump, when it exists, should classify and then choose:

```go
return transport.Respond(conn, classify.Classify(err))
```

## Panic recovery

Recovery is local to the goroutine that panics. Gin middleware does not recover a panic in another goroutine.

Both paths build `internal.panic_recovered`, keep the panic value as the cause, attach `runtime/debug.Stack()`, and log at error. The client text is the registry message, `An unexpected error occurred`.

HTTP: `recovery.Gin` is `gin.CustomRecoveryWithWriter(nil, ...)`. The nil writer skips Gin's logger. The callback logs the panic once, including the method and the URL path, then writes that JSON with status 500 when the body is not already written. A body that was written before the panic is left as it is, and the context is aborted. `gin.Recovery()` is not installed.

WebSocket:

```go
recovery.Go(log, conn, recovery.ConnContext{
    RequestID:    requestID,
    ConnectionID: connectionID,
    UserID:       userID,
}, client.ReadPump)
```

`Go` starts `fn` on a new goroutine with its own `defer recover`. On panic it logs and, when `conn` is non-nil, sends close code `4000` through `WriteControl`. It does not call `WriteJSON` after a panic. A nil connection is only logged. If the close write fails, that failure is a second log line, `websocket close after panic failed`, and the goroutine ends. The panic itself was already logged.

Every independently launched pump needs its own `recovery.Go`. A panic inside `ReadPump` is not caught by a `Go` wrapped around a different function.

## HTTP server wiring

`internal/transport/rest.New` calls `errs.MustLoad`, then registers middleware outermost first:

1. `recovery.Gin`, so a panic in later middleware is recovered.
2. `middleware.RequestID`. `X-Request-ID` is reused when it is non-empty, at most 128 characters, and contains no characters below `0x20` or `0x7f`. Otherwise the server generates 16 random bytes, hex-encoded. The id is stored with `errcore.WithRequestID` and echoed on the response. It still runs before the handler, so a handler panic log includes it.
3. `gin-contrib/logger` access log.
4. `middleware.Errors` for leftover `c.Errors`.

Future auth middleware can store the user id with `errcore.WithUserID` on the request context. The error logger reads it from there. No auth middleware exists yet.

## Adding an error

Edit only `internal/errs`.

1. Add a constant in `code.go`. The value must match the code format and must be new.
2. Add one `def(...)` line in the domain file's definition function. Use `defClose` when the WebSocket transport should close.
3. The function is already included from `Definitions`. Add it there only if the domain file is new.
4. Add the constant to `catalog` in `catalog_test.go`.

Example, inside `internal/errs/code.go` and `internal/errs/user.go`:

```go
const UserSuspended errcore.Code = "user.suspended"

func userDefinitions() []errcore.Definition {
    return []errcore.Definition{
        // existing rows...
        def(UserSuspended, errcore.CategoryAuthorization, "Account is suspended", 403, errcore.LogWarn, false, false),
    }
}
```

Callers:

```go
return errcore.New(ctx, errs.UserSuspended)
```

A duplicate code panics when `MustLoad` runs. A bad status, an empty message, a malformed code, or an invalid socket policy does the same.

Do not change `internal/packages/errcore` to add a product error.

## Service and repository use

Repositories may keep returning driver errors. The service decides the public code:

```go
row, err := s.repository.GetByID(ctx, id)
if errors.Is(err, gorm.ErrRecordNotFound) {
    return errcore.Wrap(ctx, errs.UserNotFound, err)
}
if err != nil {
    return errcore.Wrap(ctx, errs.DatabaseQueryFailed, err)
}
```

`errcore.Wrap` preserves the driver error for logs. The HTTP body says `A database operation failed` and does not include the driver text.

Safe metadata:

```go
return errcore.New(ctx, errs.UserNotFound).WithMetadata(map[string]any{
    "user_id": id,
})
```

Do not put passwords, tokens, or message bodies in metadata. Redaction is a backstop for known key names, not a guarantee for every secret.

## Tests

| Package | What it locks |
|---|---|
| `internal/packages/errcore` | construct, wrap, unwrap, `errors.Is`, `errors.As`, metadata copy, lookup, fallback, duplicate registration, invalid definitions |
| `internal/errs` | every constant is registered, and every registered code has a constant |
| `internal/packages/errcore/classify` | known, wrapped, unknown, nil, status, public message, log level, redaction, context fields, metric key |
| `internal/packages/errcore/transport` | HTTP status and JSON, unknown 500, no second write, already-written body, middleware, WebSocket event, auth close `4001`, write failure |
| `internal/packages/errcore/recovery` | HTTP panic body has no panic text, already-written body kept, WebSocket panic close `4000`, nil connection still logs, close beside a data writer |
| `internal/transport/rest` | request id reuse and replacement, server panic stays internal and sets `X-Request-ID` |
| `internal/transport/rest/middleware` | request id and leftover handler errors |

Run:

```text
go test ./internal/errs/... ./internal/packages/errcore/... ./internal/transport/rest/
go test -race ./internal/packages/errcore/transport ./internal/packages/errcore/recovery ./internal/transport/rest
```

## Current limits

- Routes, handlers, and business services are not written. They call `transport.AbortWithError` or return `errcore.Wrap` when they appear.
- Repositories still return raw GORM errors. Wrapping them is a service decision.
- There is no WebSocket server or read/write pump. `transport` and `recovery.Go` are ready for that pump.
- `ShouldAlert` is recorded and logged. No notifier is attached.
- `MetricKey` is the Prometheus-shaped value. No counter is registered.
- Trace ID is read when present. This process does not create traces.
