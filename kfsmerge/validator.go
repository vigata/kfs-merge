package kfsmerge

import (
	"encoding/json"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// ValidationError represents a validation failure.
type ValidationError struct {
	Path    string
	Message string
	Phase   ValidationPhase
}

// Error implements the error interface.
func (e ValidationError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.Phase, e.Path, e.Message)
}

// Validator validates JSON instances against a schema.
type Validator struct {
	schema *Schema
}

// NewValidator creates a new Validator for the given schema.
func NewValidator(s *Schema) *Validator {
	return &Validator{schema: s}
}

// Validate validates a JSON instance and returns the first error encountered.
func (v *Validator) Validate(instanceJSON []byte, phase ValidationPhase) error {
	var instance any
	if err := json.Unmarshal(instanceJSON, &instance); err != nil {
		return ValidationError{
			Path:    "",
			Message: fmt.Sprintf("invalid JSON: %v", err),
			Phase:   phase,
		}
	}

	err := v.schema.CompiledSchema().Validate(instance)
	if err == nil {
		return nil
	}

	return v.convertError(err, phase)
}

// ValidateValue validates an already-parsed value.
func (v *Validator) ValidateValue(instance any, phase ValidationPhase) error {
	err := v.schema.CompiledSchema().Validate(instance)
	if err == nil {
		return nil
	}
	return v.convertError(err, phase)
}

// convertError converts a jsonschema validation error to our ValidationError type.
func (v *Validator) convertError(err error, phase ValidationPhase) ValidationError {
	if validationErr, ok := err.(*jsonschema.ValidationError); ok {
		path := "/" + joinPath(validationErr.InstanceLocation)
		return ValidationError{
			Path:    path,
			Message: validationErr.Error(),
			Phase:   phase,
		}
	}

	return ValidationError{
		Path:    "",
		Message: err.Error(),
		Phase:   phase,
	}
}

// joinPath joins path segments with /.
func joinPath(segments []string) string {
	if len(segments) == 0 {
		return ""
	}
	result := segments[0]
	for _, s := range segments[1:] {
		result += "/" + s
	}
	return result
}

