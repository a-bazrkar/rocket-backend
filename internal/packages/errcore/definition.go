package errcore

// LogLevel is the severity stored on a definition.
// The classifier maps it to the logging library.
type LogLevel string

const (
	LogDebug LogLevel = "debug"
	LogInfo  LogLevel = "info"
	LogWarn  LogLevel = "warn"
	LogError LogLevel = "error"
)

// SocketPolicy tells the WebSocket transport how to present one error.
// It is a plain string so this package does not import Gorilla.
type SocketPolicy string

const (
	// SocketEvent sends an application-level error event and leaves the connection open.
	// The zero value is this policy.
	SocketEvent SocketPolicy = ""
	// SocketClose closes the connection and does not send an error event.
	SocketClose SocketPolicy = "close"
	// SocketEventClose sends an error event and then closes the connection.
	SocketEventClose SocketPolicy = "event_close"
)

const (
	// CloseCodePanic is the private-use close code for a recovered panic.
	CloseCodePanic = 4000
	// CloseCodeAuth is the private-use close code for an authentication failure
	// whose definition opts into SocketClose.
	CloseCodeAuth = 4001
)

// Definition is the public contract for one error code.
// Message is safe to return to clients. HTTPStatus is a numeric status, not an HTTP dependency.
// Socket and CloseCode are optional WebSocket instructions. An empty Socket keeps the connection open.
type Definition struct {
	Code        Code
	Category    Category
	Message     string
	HTTPStatus  int
	Retryable   bool
	LogLevel    LogLevel
	ShouldAlert bool
	Socket      SocketPolicy
	CloseCode   int
}
