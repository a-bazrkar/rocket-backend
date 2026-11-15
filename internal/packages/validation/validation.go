package validation

import "github.com/go-playground/validator/v10"

// Validator checks struct tags from go-playground/validator.
type Validator struct {
	validate *validator.Validate
}

// New creates a new Validator.
func New() *Validator {
	return &Validator{validate: validator.New()}
}

// Struct validates s using its validate tags.
func (v *Validator) Struct(s any) error {
	return v.validate.Struct(s)
}

// i have created this package for future custom validation rules
// for now, we are using the default validation rules from go-playground/validator
// but we can add custom validation rules here in the future
