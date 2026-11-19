package classify

import (
	stderrors "errors"

	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore"
)

// Classify maps any error to a public classification.
// A nil error returns a zero classification and is not logged.
// Unknown errors use the internal.unexpected_error definition.
func Classify(err error) Classification {
	if err == nil {
		return Classification{}
	}
	var app *errcore.Error
	if !stderrors.As(err, &app) || app == nil {
		return fromDefinition(errcore.Fallback(), err, nil, errcore.Operation{})
	}
	op := app.Operation()
	def, ok := errcore.Lookup(app.Code)
	if !ok {
		meta := redact(app.Metadata)
		if meta == nil {
			meta = map[string]any{}
		}
		meta["unregistered_code"] = string(app.Code)
		return fromDefinition(errcore.Fallback(), err, meta, op)
	}
	return fromDefinition(def, err, redact(app.Metadata), op)
}

func fromDefinition(def errcore.Definition, cause error, meta map[string]any, op errcore.Operation) Classification {
	return Classification{
		Code:        def.Code,
		Category:    def.Category,
		Message:     def.Message,
		HTTPStatus:  def.HTTPStatus,
		LogLevel:    def.LogLevel,
		ShouldLog:   true,
		ShouldAlert: def.ShouldAlert,
		Retryable:   def.Retryable,
		Socket:      def.Socket,
		CloseCode:   def.CloseCode,
		Cause:       cause,
		Metadata:    meta,
		Operation:   op,
	}
}
