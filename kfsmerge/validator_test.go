package kfsmerge

import (
	"encoding/json"
	"testing"
)

func TestValidate(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"count": {"type": "integer"}
		},
		"required": ["name"]
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	tests := []struct {
		name      string
		instance  string
		wantError bool
	}{
		{"valid", `{"name": "test", "count": 5}`, false},
		{"valid minimal", `{"name": "test"}`, false},
		{"missing required", `{"count": 5}`, true},
		{"wrong type", `{"name": 123}`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.Validate([]byte(tt.instance))
			if tt.wantError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestMergeValidationFailure(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		},
		"required": ["name"]
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// Invalid A (missing required field)
	a := []byte(`{}`)
	b := []byte(`{"name": "template"}`)

	_, err = s.Merge(a, b)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestMergeWithOptionsSkipValidation(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		},
		"required": ["name"]
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// A is missing required field
	a := []byte(`{"extra": "value"}`)
	b := []byte(`{"name": "template"}`)

	// Without SkipValidateA, this should fail
	_, err = s.Merge(a, b)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	// With SkipValidateA, this should succeed
	opts := MergeOptions{SkipValidateA: true}
	result, err := s.MergeWithOptions(a, b, opts)
	if err != nil {
		t.Fatalf("MergeWithOptions failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if got["name"] != "template" {
		t.Errorf("name = %v, want 'template'", got["name"])
	}
	if got["extra"] != "value" {
		t.Errorf("extra = %v, want 'value'", got["extra"])
	}
}

func TestMergeWithOptionsSkipResultValidation(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"count": {"type": "integer", "minimum": 10}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// A has count=5 (below minimum), B has count=15 (valid)
	// After merge, A's count=5 wins, which violates minimum
	a := []byte(`{"count": 5}`)
	b := []byte(`{"name": "test", "count": 15}`)

	// With SkipValidateA and without SkipValidateResult, result validation should fail
	opts := MergeOptions{SkipValidateA: true}
	_, err = s.MergeWithOptions(a, b, opts)
	if err == nil {
		t.Fatal("expected validation error for minimum violation, got nil")
	}

	// With both SkipValidateA and SkipValidateResult, should succeed
	opts = MergeOptions{SkipValidateA: true, SkipValidateResult: true}
	result, err := s.MergeWithOptions(a, b, opts)
	if err != nil {
		t.Fatalf("MergeWithOptions failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if got["count"] != float64(5) {
		t.Errorf("count = %v, want 5", got["count"])
	}
}

