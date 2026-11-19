package errs

import "github.com/a-bazrkar/rocket-backend/internal/packages/errcore"

func def(code errcore.Code, category errcore.Category, message string, status int, level errcore.LogLevel, retryable, alert bool) errcore.Definition {
	return errcore.Definition{
		Code:        code,
		Category:    category,
		Message:     message,
		HTTPStatus:  status,
		Retryable:   retryable,
		LogLevel:    level,
		ShouldAlert: alert,
	}
}

// defClose is def with a WebSocket close and no error event.
func defClose(code errcore.Code, category errcore.Category, message string, status int, level errcore.LogLevel, retryable, alert bool, closeCode int) errcore.Definition {
	d := def(code, category, message, status, level, retryable, alert)
	d.Socket = errcore.SocketClose
	d.CloseCode = closeCode
	return d
}
