package recovery

import (
	"context"
	"fmt"

	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore/classify"
	"github.com/a-bazrkar/rocket-backend/internal/packages/logger"
)

func report(log logger.Logger, value any, stack []byte, ctx context.Context, scope classify.Scope) classify.Classification {
	err := errcore.Wrap(ctx, errcore.CodeInternalPanicRecovered, panicCause(value))
	class := classify.Classify(err)
	class.Stack = string(stack)
	classify.Log(log, class, ctx, scope)
	return class
}

func panicCause(value any) error {
	if err, ok := value.(error); ok && err != nil {
		return err
	}
	return fmt.Errorf("%v", value)
}
