// Package validate provides JSON Schema validation with detailed error reporting.
package validate

import (
	"encoding/json"
	"fmt"

	"github.com/nbcuni/kfs-flow-merge/schema"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Phase indicates which stage of the merge process a validation error occurred.
type Phase string

const (
	// PhaseValidateA indicates validation of instance A (API request).
	PhaseValidateA Phase = "validate_a"
	// PhaseValidateB indicates validation of instance B (template).
	PhaseValidateB Phase = "validate_b"
	// PhaseValidateResult indicates validation of the merge result.
	PhaseValidateResult Phase = "validate_result"
)

// Error represents a validation failure.
type Error struct {
	// Path is the JSON pointer to the failing location (e.g., "/config/timeout").
	Path string
	// Message describes the validation failure.
	Message string
	// Phase indicates when this error occurred.
	Phase Phase
}

// Error implements the error interface.
func (e Error) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.Phase, e.Path, e.Message)
}

// Validator validates JSON instances against a schema.
type Validator struct {
	schema *schema.Schema
}

// New creates a new Validator for the given schema.
func New(s *schema.Schema) *Validator {
	return &Validator{schema: s}
}

// Validate validates a JSON instance and returns the first error encountered.
// Returns nil if validation succeeds.
func (v *Validator) Validate(instanceJSON []byte, phase Phase) error {
	var instance any
	if err := json.Unmarshal(instanceJSON, &instance); err != nil {
		return Error{
			Path:    "",
			Message: fmt.Sprintf("invalid JSON: %v", err),
			Phase:   phase,
		}
	}

	err := v.schema.CompiledSchema().Validate(instance)
	if err == nil {
		return nil
	}

	// Convert jsonschema error to our Error type
	return v.convertError(err, phase)
}

// ValidateValue validates an already-parsed value.
func (v *Validator) ValidateValue(instance any, phase Phase) error {
	err := v.schema.CompiledSchema().Validate(instance)
	if err == nil {
		return nil
	}
	return v.convertError(err, phase)
}

// convertError converts a jsonschema validation error to our Error type.
func (v *Validator) convertError(err error, phase Phase) Error {
	// The jsonschema library returns detailed errors
	if validationErr, ok := err.(*jsonschema.ValidationError); ok {
		// InstanceLocation is []string, join with /
		path := "/" + joinPath(validationErr.InstanceLocation)
		return Error{
			Path:    path,
			Message: validationErr.Error(),
			Phase:   phase,
		}
	}

	// Fallback for other error types
	return Error{
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
