// Package kfsmerge provides JSON Schema-based merging of JSON instances.
//
// It validates two JSON instances (A and B) against a schema, merges them
// according to rules embedded in the schema using x-kfs-merge extensions,
// and validates the result.
//
// When merging using Merge(a, b):
//   - A (first parameter) is the request/override instance (typically API request or user input)
//   - B (second parameter) is the base/template instance (typically defaults or template configuration)
//   - By default, A takes precedence over B (request overrides base)
package kfsmerge

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/nbcuni/kfs-flow-merge/merge"
	"github.com/nbcuni/kfs-flow-merge/schema"
	"github.com/nbcuni/kfs-flow-merge/validate"
)

// Schema represents a loaded JSON Schema with merge extensions.
type Schema struct {
	internal *schema.Schema
}

// MergeOptions controls the merge and validation behavior.
type MergeOptions struct {
	// SkipValidateA skips validation of instance A.
	SkipValidateA bool
	// SkipValidateB skips validation of instance B.
	SkipValidateB bool
	// SkipValidateResult skips validation of the merged result.
	SkipValidateResult bool
}

// DefaultMergeOptions returns the default options (all validations enabled).
func DefaultMergeOptions() MergeOptions {
	return MergeOptions{}
}

// LoadSchema parses a JSON Schema with x-kfs-merge extensions from bytes.
func LoadSchema(schemaJSON []byte) (*Schema, error) {
	s, err := schema.Load(schemaJSON)
	if err != nil {
		return nil, err
	}
	return &Schema{internal: s}, nil
}

// LoadSchemaFromFile loads a JSON Schema from a file path.
func LoadSchemaFromFile(path string) (*Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}
	return LoadSchema(data)
}

// LoadSchemaFromURL loads a JSON Schema from a URL.
func LoadSchemaFromURL(url string) (*Schema, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch schema from URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch schema: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema response: %w", err)
	}
	return LoadSchema(data)
}

// LoadSchemaFromSource loads a schema from a file path, URL, or raw JSON.
// It automatically detects the source type based on the input.
func LoadSchemaFromSource(source string) (*Schema, error) {
	// Check if it's a URL
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		return LoadSchemaFromURL(source)
	}

	// Check if it looks like JSON (starts with { after trimming whitespace)
	trimmed := strings.TrimSpace(source)
	if strings.HasPrefix(trimmed, "{") {
		return LoadSchema([]byte(source))
	}

	// Assume it's a file path
	return LoadSchemaFromFile(source)
}

// Merge validates instances A and B, merges them, and validates the result.
// This is equivalent to MergeWithOptions with default options.
func (s *Schema) Merge(a, b []byte) ([]byte, error) {
	return s.MergeWithOptions(a, b, DefaultMergeOptions())
}

// MergeWithOptions merges A into B with configurable validation behavior.
//
// The merge process:
//  1. Validate A against the schema (unless SkipValidateA is set)
//  2. Validate B against the schema (unless SkipValidateB is set)
//  3. Merge A into B according to x-kfs-merge rules
//  4. Validate the result (unless SkipValidateResult is set)
//
// Returns the merged instance as JSON bytes, or an error if any step fails.
func (s *Schema) MergeWithOptions(a, b []byte, opts MergeOptions) ([]byte, error) {
	validator := validate.New(s.internal)

	// Step 1: Validate A
	if !opts.SkipValidateA {
		if err := validator.Validate(a, validate.PhaseValidateA); err != nil {
			return nil, fmt.Errorf("instance A validation failed: %w", err)
		}
	}

	// Step 2: Validate B
	if !opts.SkipValidateB {
		if err := validator.Validate(b, validate.PhaseValidateB); err != nil {
			return nil, fmt.Errorf("instance B validation failed: %w", err)
		}
	}

	// Parse instances
	var aVal, bVal any
	if err := json.Unmarshal(a, &aVal); err != nil {
		return nil, fmt.Errorf("failed to parse instance A: %w", err)
	}
	if err := json.Unmarshal(b, &bVal); err != nil {
		return nil, fmt.Errorf("failed to parse instance B: %w", err)
	}

	// Step 3: Merge
	merger := merge.New(s.internal)
	result, err := merger.Merge(aVal, bVal)
	if err != nil {
		return nil, fmt.Errorf("merge failed: %w", err)
	}

	// Step 4: Validate result
	if !opts.SkipValidateResult {
		if err := validator.ValidateValue(result, validate.PhaseValidateResult); err != nil {
			return nil, fmt.Errorf("result validation failed: %w", err)
		}
	}

	// Marshal result
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return resultJSON, nil
}

// MergeToValue is like Merge but returns the result as a Go value instead of JSON bytes.
func (s *Schema) MergeToValue(a, b []byte) (any, error) {
	return s.MergeToValueWithOptions(a, b, DefaultMergeOptions())
}

// MergeToValueWithOptions is like MergeWithOptions but returns the result as a Go value.
func (s *Schema) MergeToValueWithOptions(a, b []byte, opts MergeOptions) (any, error) {
	validator := validate.New(s.internal)

	// Step 1: Validate A
	if !opts.SkipValidateA {
		if err := validator.Validate(a, validate.PhaseValidateA); err != nil {
			return nil, fmt.Errorf("instance A validation failed: %w", err)
		}
	}

	// Step 2: Validate B
	if !opts.SkipValidateB {
		if err := validator.Validate(b, validate.PhaseValidateB); err != nil {
			return nil, fmt.Errorf("instance B validation failed: %w", err)
		}
	}

	// Parse instances
	var aVal, bVal any
	if err := json.Unmarshal(a, &aVal); err != nil {
		return nil, fmt.Errorf("failed to parse instance A: %w", err)
	}
	if err := json.Unmarshal(b, &bVal); err != nil {
		return nil, fmt.Errorf("failed to parse instance B: %w", err)
	}

	// Step 3: Merge
	merger := merge.New(s.internal)
	result, err := merger.Merge(aVal, bVal)
	if err != nil {
		return nil, fmt.Errorf("merge failed: %w", err)
	}

	// Step 4: Validate result
	if !opts.SkipValidateResult {
		if err := validator.ValidateValue(result, validate.PhaseValidateResult); err != nil {
			return nil, fmt.Errorf("result validation failed: %w", err)
		}
	}

	return result, nil
}

// Validate validates a JSON instance against the schema.
func (s *Schema) Validate(instanceJSON []byte) error {
	validator := validate.New(s.internal)
	return validator.Validate(instanceJSON, validate.PhaseValidateA)
}
