package kfsmerge

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

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
func LoadSchemaFromSource(source string) (*Schema, error) {
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		return LoadSchemaFromURL(source)
	}

	trimmed := strings.TrimSpace(source)
	if strings.HasPrefix(trimmed, "{") {
		return LoadSchema([]byte(source))
	}

	return LoadSchemaFromFile(source)
}

// Merge validates instances A and B, merges them, and validates the result.
func (s *Schema) Merge(a, b []byte) ([]byte, error) {
	return s.MergeWithOptions(a, b, DefaultMergeOptions())
}

// MergeWithOptions merges A into B with configurable validation behavior.
func (s *Schema) MergeWithOptions(a, b []byte, opts MergeOptions) ([]byte, error) {
	validator := NewValidator(s)

	if !opts.SkipValidateA {
		if err := validator.Validate(a, PhaseValidateA); err != nil {
			return nil, fmt.Errorf("instance A validation failed: %w", err)
		}
	}

	if !opts.SkipValidateB {
		if err := validator.Validate(b, PhaseValidateB); err != nil {
			return nil, fmt.Errorf("instance B validation failed: %w", err)
		}
	}

	var aVal, bVal any
	if err := json.Unmarshal(a, &aVal); err != nil {
		return nil, fmt.Errorf("failed to parse instance A: %w", err)
	}
	if err := json.Unmarshal(b, &bVal); err != nil {
		return nil, fmt.Errorf("failed to parse instance B: %w", err)
	}

	merger := NewMerger(s)

	// Apply defaults if enabled: merge(A, merge(B, defaults))
	if s.shouldApplyDefaults(opts) {
		defaults := s.ExtractDefaults()
		if defaults != nil {
			// First merge B into defaults
			bWithDefaults, err := merger.Merge(bVal, defaults)
			if err != nil {
				return nil, fmt.Errorf("failed to apply defaults to B: %w", err)
			}
			bVal = bWithDefaults
		}
	}

	result, err := merger.Merge(aVal, bVal)
	if err != nil {
		return nil, fmt.Errorf("merge failed: %w", err)
	}

	if !opts.SkipValidateResult {
		if err := validator.ValidateValue(result, PhaseValidateResult); err != nil {
			return nil, fmt.Errorf("result validation failed: %w", err)
		}
	}

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
	validator := NewValidator(s)

	if !opts.SkipValidateA {
		if err := validator.Validate(a, PhaseValidateA); err != nil {
			return nil, fmt.Errorf("instance A validation failed: %w", err)
		}
	}

	if !opts.SkipValidateB {
		if err := validator.Validate(b, PhaseValidateB); err != nil {
			return nil, fmt.Errorf("instance B validation failed: %w", err)
		}
	}

	var aVal, bVal any
	if err := json.Unmarshal(a, &aVal); err != nil {
		return nil, fmt.Errorf("failed to parse instance A: %w", err)
	}
	if err := json.Unmarshal(b, &bVal); err != nil {
		return nil, fmt.Errorf("failed to parse instance B: %w", err)
	}

	merger := NewMerger(s)

	// Apply defaults if enabled: merge(A, merge(B, defaults))
	if s.shouldApplyDefaults(opts) {
		defaults := s.ExtractDefaults()
		if defaults != nil {
			// First merge B into defaults
			bWithDefaults, err := merger.Merge(bVal, defaults)
			if err != nil {
				return nil, fmt.Errorf("failed to apply defaults to B: %w", err)
			}
			bVal = bWithDefaults
		}
	}

	result, err := merger.Merge(aVal, bVal)
	if err != nil {
		return nil, fmt.Errorf("merge failed: %w", err)
	}

	if !opts.SkipValidateResult {
		if err := validator.ValidateValue(result, PhaseValidateResult); err != nil {
			return nil, fmt.Errorf("result validation failed: %w", err)
		}
	}

	return result, nil
}

// Validate validates a JSON instance against the schema.
func (s *Schema) Validate(instanceJSON []byte) error {
	validator := NewValidator(s)
	return validator.Validate(instanceJSON, PhaseValidateA)
}

// shouldApplyDefaults determines if defaults should be applied based on
// schema config and merge options. Options override schema setting.
func (s *Schema) shouldApplyDefaults(opts MergeOptions) bool {
	if opts.ApplyDefaults != nil {
		return *opts.ApplyDefaults
	}
	return s.globalConfig.ApplyDefaults
}
