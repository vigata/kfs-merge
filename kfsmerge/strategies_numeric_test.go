package kfsmerge

import (
	"encoding/json"
	"testing"
)

// =============================================================================
// Numeric Strategy Tests
// =============================================================================

func TestMergeNumericStrategies(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"sum": {
				"type": "integer",
				"x-kfs-merge": {"strategy": "numeric", "operation": "sum"}
			},
			"max": {
				"type": "integer",
				"x-kfs-merge": {"strategy": "numeric", "operation": "max"}
			},
			"min": {
				"type": "integer",
				"x-kfs-merge": {"strategy": "numeric", "operation": "min"}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	a := []byte(`{"sum": 10, "max": 5, "min": 5}`)
	b := []byte(`{"sum": 20, "max": 15, "min": 15}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if got["sum"] != float64(30) {
		t.Errorf("sum = %v, want 30", got["sum"])
	}
	if got["max"] != float64(15) {
		t.Errorf("max = %v, want 15", got["max"])
	}
	if got["min"] != float64(5) {
		t.Errorf("min = %v, want 5", got["min"])
	}
}

// TestMergeNumericDefaultOperation tests that default operation is "sum".
func TestMergeNumericDefaultOperation(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"count": {
				"type": "integer",
				"x-kfs-merge": {"strategy": "numeric"}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	a := []byte(`{"count": 10}`)
	b := []byte(`{"count": 5}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	// Default operation is sum: 10 + 5 = 15
	if got["count"] != float64(15) {
		t.Errorf("count = %v, want 15 (default operation is sum)", got["count"])
	}
}

// TestMergeNumericWithFloats tests numeric operations with float values.
func TestMergeNumericWithFloats(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"sum": {
				"type": "number",
				"x-kfs-merge": {"strategy": "numeric", "operation": "sum"}
			},
			"max": {
				"type": "number",
				"x-kfs-merge": {"strategy": "numeric", "operation": "max"}
			},
			"min": {
				"type": "number",
				"x-kfs-merge": {"strategy": "numeric", "operation": "min"}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	a := []byte(`{"sum": 1.5, "max": 2.7, "min": 1.2}`)
	b := []byte(`{"sum": 2.5, "max": 3.1, "min": 0.8}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	// sum: 1.5 + 2.5 = 4.0
	if got["sum"] != float64(4.0) {
		t.Errorf("sum = %v, want 4.0", got["sum"])
	}
	// max: max(2.7, 3.1) = 3.1
	if got["max"] != float64(3.1) {
		t.Errorf("max = %v, want 3.1", got["max"])
	}
	// min: min(1.2, 0.8) = 0.8
	if got["min"] != float64(0.8) {
		t.Errorf("min = %v, want 0.8", got["min"])
	}
}

// TestMergeNumericWithNegatives tests numeric operations with negative numbers.
func TestMergeNumericWithNegatives(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"sum": {
				"type": "integer",
				"x-kfs-merge": {"strategy": "numeric", "operation": "sum"}
			},
			"max": {
				"type": "integer",
				"x-kfs-merge": {"strategy": "numeric", "operation": "max"}
			},
			"min": {
				"type": "integer",
				"x-kfs-merge": {"strategy": "numeric", "operation": "min"}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	a := []byte(`{"sum": -10, "max": -5, "min": -15}`)
	b := []byte(`{"sum": 5, "max": -10, "min": -5}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	// sum: -10 + 5 = -5
	if got["sum"] != float64(-5) {
		t.Errorf("sum = %v, want -5", got["sum"])
	}
	// max: max(-5, -10) = -5
	if got["max"] != float64(-5) {
		t.Errorf("max = %v, want -5", got["max"])
	}
	// min: min(-15, -5) = -15
	if got["min"] != float64(-15) {
		t.Errorf("min = %v, want -15", got["min"])
	}
}

// TestMergeNumericWithZero tests numeric operations with zero values.
func TestMergeNumericWithZero(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"sum": {
				"type": "integer",
				"x-kfs-merge": {"strategy": "numeric", "operation": "sum"}
			},
			"max": {
				"type": "integer",
				"x-kfs-merge": {"strategy": "numeric", "operation": "max"}
			},
			"min": {
				"type": "integer",
				"x-kfs-merge": {"strategy": "numeric", "operation": "min"}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	a := []byte(`{"sum": 0, "max": 0, "min": 0}`)
	b := []byte(`{"sum": 10, "max": -5, "min": 5}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	// sum: 0 + 10 = 10
	if got["sum"] != float64(10) {
		t.Errorf("sum = %v, want 10", got["sum"])
	}
	// max: max(0, -5) = 0
	if got["max"] != float64(0) {
		t.Errorf("max = %v, want 0", got["max"])
	}
	// min: min(0, 5) = 0
	if got["min"] != float64(0) {
		t.Errorf("min = %v, want 0", got["min"])
	}
}

// TestMergeNumericOneMissing tests numeric when one value is missing.
func TestMergeNumericOneMissing(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"countA": {
				"type": "integer",
				"x-kfs-merge": {"strategy": "numeric", "operation": "sum"}
			},
			"countB": {
				"type": "integer",
				"x-kfs-merge": {"strategy": "numeric", "operation": "sum"}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// A has countA only, B has countB only
	a := []byte(`{"countA": 10}`)
	b := []byte(`{"countB": 20}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	// When only A has value, use A's value
	if got["countA"] != float64(10) {
		t.Errorf("countA = %v, want 10", got["countA"])
	}
	// When only B has value, use B's value
	if got["countB"] != float64(20) {
		t.Errorf("countB = %v, want 20", got["countB"])
	}
}
