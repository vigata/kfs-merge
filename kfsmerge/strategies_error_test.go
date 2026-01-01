package kfsmerge

import (
	"testing"
)

// =============================================================================
// Error Condition Tests
// =============================================================================

// TestConcatStrategyErrorOnNonArray tests that concat strategy returns error for non-arrays.
func TestConcatStrategyErrorOnNonArray(t *testing.T) {
	// Use MergeToValueWithOptions to skip validation and test the merge logic directly
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"value": {
				"x-kfs-merge": {"strategy": "concat"}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// Both have strings instead of arrays - skip validation to reach the merge logic
	a := []byte(`{"value": "string"}`)
	b := []byte(`{"value": "another"}`)

	opts := MergeOptions{SkipValidateA: true, SkipValidateB: true, SkipValidateResult: true}
	_, err = s.MergeWithOptions(a, b, opts)
	if err == nil {
		t.Fatal("expected error for concat on non-arrays, got nil")
	}
}

// TestMergeByDiscriminatorErrorOnNonArray tests that mergeByDiscriminator returns error for non-arrays.
func TestMergeByDiscriminatorErrorOnNonArray(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"items": {
				"x-kfs-merge": {"strategy": "mergeByDiscriminator", "discriminatorField": "type"}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// Both have objects instead of arrays
	a := []byte(`{"items": {"key": "value"}}`)
	b := []byte(`{"items": {"key": "other"}}`)

	opts := MergeOptions{SkipValidateA: true, SkipValidateB: true, SkipValidateResult: true}
	_, err = s.MergeWithOptions(a, b, opts)
	if err == nil {
		t.Fatal("expected error for mergeByDiscriminator on non-arrays, got nil")
	}
}

// TestNumericStrategyErrorOnNonNumbers tests that numeric strategy returns error for non-numbers.
func TestNumericStrategyErrorOnNonNumbers(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"count": {
				"x-kfs-merge": {"strategy": "numeric", "operation": "sum"}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// Both have strings instead of numbers
	a := []byte(`{"count": "not-a-number"}`)
	b := []byte(`{"count": "also-not-a-number"}`)

	opts := MergeOptions{SkipValidateA: true, SkipValidateB: true, SkipValidateResult: true}
	_, err = s.MergeWithOptions(a, b, opts)
	if err == nil {
		t.Fatal("expected error for numeric operation on non-numbers, got nil")
	}
}

// TestNumericStrategyInvalidOperation tests that numeric strategy returns error for unknown operation.
func TestNumericStrategyInvalidOperation(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"count": {
				"type": "integer",
				"x-kfs-merge": {"strategy": "numeric", "operation": "invalid"}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	a := []byte(`{"count": 10}`)
	b := []byte(`{"count": 5}`)

	_, err = s.Merge(a, b)
	if err == nil {
		t.Fatal("expected error for invalid numeric operation, got nil")
	}
}

