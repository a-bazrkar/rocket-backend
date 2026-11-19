package errcore

import (
	"fmt"
	"regexp"
	"slices"
	"sync/atomic"
)

// codePattern is the stable public format: lowercase segments separated by dots.
var codePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`)

// Registry is an immutable set of error definitions.
// Build one with NewRegistry and publish it with SetDefault.
// The map is not modified after NewRegistry returns.
type Registry struct {
	byCode map[Code]Definition
}

// NewRegistry validates defs and returns a registry that holds them.
// A duplicate code or an invalid definition returns an error.
// The returned registry cannot be changed.
func NewRegistry(defs []Definition) (*Registry, error) {
	if err := Validate(defs); err != nil {
		return nil, err
	}
	byCode := make(map[Code]Definition, len(defs))
	for _, def := range defs {
		byCode[def.Code] = def
	}
	return &Registry{byCode: byCode}, nil
}

// Lookup returns the definition for code.
func (r *Registry) Lookup(code Code) (Definition, bool) {
	if r == nil {
		return Definition{}, false
	}
	def, ok := r.byCode[code]
	return def, ok
}

// Codes returns the registered codes in lexical order.
func (r *Registry) Codes() []Code {
	if r == nil {
		return nil
	}
	out := make([]Code, 0, len(r.byCode))
	for code := range r.byCode {
		out = append(out, code)
	}
	slices.Sort(out)
	return out
}

// defaultRegistry is the process registry installed by SetDefault.
// Lookups load the pointer. The map behind it is never written.
var defaultRegistry atomic.Pointer[Registry]

// SetDefault installs r as the process registry used by Lookup, New, and Classify.
// A nil registry panics. r is not modified by later calls.
func SetDefault(r *Registry) {
	if r == nil {
		panic("errors: nil registry")
	}
	defaultRegistry.Store(r)
}

// Lookup returns the definition for code from the process registry.
func Lookup(code Code) (Definition, bool) {
	return defaultRegistry.Load().Lookup(code)
}

// Codes returns the process registry codes in lexical order.
func Codes() []Code {
	return defaultRegistry.Load().Codes()
}

// Fallback is the public definition used when an error is unknown.
func Fallback() Definition {
	if def, ok := Lookup(CodeInternalUnexpectedError); ok {
		return def
	}
	return Definition{
		Code:        CodeInternalUnexpectedError,
		Category:    CategoryInternal,
		Message:     "An unexpected error occurred",
		HTTPStatus:  500,
		LogLevel:    LogError,
		ShouldAlert: true,
	}
}

// Validate reports duplicate codes and definitions that break the public contract.
// It does not mutate a registry.
func Validate(defs []Definition) error {
	seen := make(map[Code]struct{}, len(defs))
	for i, def := range defs {
		if err := validateOne(def); err != nil {
			return fmt.Errorf("errors: definition %d: %w", i, err)
		}
		if _, ok := seen[def.Code]; ok {
			return fmt.Errorf("errors: duplicate code %q", def.Code)
		}
		seen[def.Code] = struct{}{}
	}
	return nil
}

func validateOne(def Definition) error {
	if !codePattern.MatchString(string(def.Code)) {
		return fmt.Errorf("invalid code %q", def.Code)
	}
	if !validCategory(def.Category) {
		return fmt.Errorf("invalid category %q for %s", def.Category, def.Code)
	}
	if def.Message == "" {
		return fmt.Errorf("empty message for %s", def.Code)
	}
	if def.HTTPStatus < 400 || def.HTTPStatus > 599 {
		return fmt.Errorf("http status %d for %s", def.HTTPStatus, def.Code)
	}
	if !validLevel(def.LogLevel) {
		return fmt.Errorf("invalid log level %q for %s", def.LogLevel, def.Code)
	}
	if err := validateSocket(def); err != nil {
		return err
	}
	return nil
}

func validateSocket(def Definition) error {
	switch def.Socket {
	case SocketEvent:
		if def.CloseCode != 0 {
			return fmt.Errorf("close code %d set without a close policy for %s", def.CloseCode, def.Code)
		}
	case SocketClose, SocketEventClose:
		if def.CloseCode < 4000 || def.CloseCode > 4999 {
			return fmt.Errorf("close code %d for %s is outside 4000-4999", def.CloseCode, def.Code)
		}
	default:
		return fmt.Errorf("invalid socket policy %q for %s", def.Socket, def.Code)
	}
	return nil
}

func validCategory(category Category) bool {
	switch category {
	case CategoryValidation, CategoryAuthentication, CategoryAuthorization,
		CategoryNotFound, CategoryConflict, CategoryRateLimit,
		CategoryExternal, CategoryDatabase, CategoryInternal:
		return true
	default:
		return false
	}
}

func validLevel(level LogLevel) bool {
	switch level {
	case LogDebug, LogInfo, LogWarn, LogError:
		return true
	default:
		return false
	}
}
