package kfsmerge

import (
	"encoding/json"
	"testing"
)

// =============================================================================
// Null Handling Tests
// =============================================================================

func TestMergeNullHandlingPreserve(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"value": {
				"anyOf": [{"type": "string"}, {"type": "null"}],
				"x-kfs-merge": {"nullHandling": "preserve"}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// A has null, B has value - with preserve, null wins
	a := []byte(`{"value": null}`)
	b := []byte(`{"value": "from-base"}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if got["value"] != nil {
		t.Errorf("value = %v, want nil (preserve null)", got["value"])
	}
}

func TestMergeNullHandlingAsAbsent(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"value": {
				"anyOf": [{"type": "string"}, {"type": "null"}],
				"x-kfs-merge": {"nullHandling": "asAbsent"}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// A has null, B has value - with asAbsent, B's value preserved
	a := []byte(`{"value": null}`)
	b := []byte(`{"value": "from-base"}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if got["value"] != "from-base" {
		t.Errorf("value = %v, want 'from-base' (asAbsent null handling)", got["value"])
	}
}

// TestNullHandlingWithKeepBase tests null handling modes with keepBase strategy.
func TestNullHandlingWithKeepBase(t *testing.T) {
	tests := []struct {
		name         string
		nullHandling string
		aValue       string
		bValue       string
		expectValue  any
	}{
		{
			name:         "asValue with null A",
			nullHandling: "asValue",
			aValue:       "null",
			bValue:       `"base"`,
			expectValue:  "base", // keepBase always uses B
		},
		{
			name:         "asAbsent with null A",
			nullHandling: "asAbsent",
			aValue:       "null",
			bValue:       `"base"`,
			expectValue:  "base", // keepBase always uses B
		},
		{
			name:         "preserve with null B",
			nullHandling: "preserve",
			aValue:       `"api"`,
			bValue:       "null",
			expectValue:  nil, // keepBase uses B's null
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schemaJSON := []byte(`{
				"$schema": "https://json-schema.org/draft/2020-12/schema",
				"type": "object",
				"properties": {
					"value": {
						"anyOf": [{"type": "string"}, {"type": "null"}],
						"x-kfs-merge": {"strategy": "keepBase", "nullHandling": "` + tt.nullHandling + `"}
					}
				}
			}`)

			s, err := LoadSchema(schemaJSON)
			if err != nil {
				t.Fatalf("LoadSchema failed: %v", err)
			}

			a := []byte(`{"value": ` + tt.aValue + `}`)
			b := []byte(`{"value": ` + tt.bValue + `}`)

			result, err := s.Merge(a, b)
			if err != nil {
				t.Fatalf("Merge failed: %v", err)
			}

			var got map[string]any
			if err := json.Unmarshal(result, &got); err != nil {
				t.Fatalf("failed to unmarshal result: %v", err)
			}

			if got["value"] != tt.expectValue {
				t.Errorf("value = %v, want %v", got["value"], tt.expectValue)
			}
		})
	}
}

// TestNullHandlingWithKeepRequest tests null handling modes with keepRequest strategy.
func TestNullHandlingWithKeepRequest(t *testing.T) {
	tests := []struct {
		name         string
		nullHandling string
		aValue       string
		bValue       string
		expectValue  any
	}{
		{
			name:         "asValue with null A",
			nullHandling: "asValue",
			aValue:       "null",
			bValue:       `"base"`,
			expectValue:  nil, // keepRequest uses A's null
		},
		{
			name:         "asAbsent with null A",
			nullHandling: "asAbsent",
			aValue:       "null",
			bValue:       `"base"`,
			expectValue:  nil, // keepRequest always uses A
		},
		{
			name:         "preserve with value A",
			nullHandling: "preserve",
			aValue:       `"api"`,
			bValue:       "null",
			expectValue:  "api", // keepRequest uses A's value
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schemaJSON := []byte(`{
				"$schema": "https://json-schema.org/draft/2020-12/schema",
				"type": "object",
				"properties": {
					"value": {
						"anyOf": [{"type": "string"}, {"type": "null"}],
						"x-kfs-merge": {"strategy": "keepRequest", "nullHandling": "` + tt.nullHandling + `"}
					}
				}
			}`)

			s, err := LoadSchema(schemaJSON)
			if err != nil {
				t.Fatalf("LoadSchema failed: %v", err)
			}

			a := []byte(`{"value": ` + tt.aValue + `}`)
			b := []byte(`{"value": ` + tt.bValue + `}`)

			result, err := s.Merge(a, b)
			if err != nil {
				t.Fatalf("Merge failed: %v", err)
			}

			var got map[string]any
			if err := json.Unmarshal(result, &got); err != nil {
				t.Fatalf("failed to unmarshal result: %v", err)
			}

			if got["value"] != tt.expectValue {
				t.Errorf("value = %v, want %v", got["value"], tt.expectValue)
			}
		})
	}
}

// TestNullHandlingWithReplace tests null handling modes with replace strategy.
func TestNullHandlingWithReplace(t *testing.T) {
	tests := []struct {
		name         string
		nullHandling string
		aValue       string
		bValue       string
		expectValue  any
	}{
		{
			name:         "asValue with null A",
			nullHandling: "asValue",
			aValue:       "null",
			bValue:       `"base"`,
			expectValue:  "base", // replace falls back to B when A is null
		},
		{
			name:         "preserve with value A",
			nullHandling: "preserve",
			aValue:       `"api"`,
			bValue:       "null",
			expectValue:  "api", // replace uses A when present
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schemaJSON := []byte(`{
				"$schema": "https://json-schema.org/draft/2020-12/schema",
				"type": "object",
				"properties": {
					"value": {
						"anyOf": [{"type": "string"}, {"type": "null"}],
						"x-kfs-merge": {"strategy": "replace", "nullHandling": "` + tt.nullHandling + `"}
					}
				}
			}`)

			s, err := LoadSchema(schemaJSON)
			if err != nil {
				t.Fatalf("LoadSchema failed: %v", err)
			}

			a := []byte(`{"value": ` + tt.aValue + `}`)
			b := []byte(`{"value": ` + tt.bValue + `}`)

			result, err := s.Merge(a, b)
			if err != nil {
				t.Fatalf("Merge failed: %v", err)
			}

			var got map[string]any
			if err := json.Unmarshal(result, &got); err != nil {
				t.Fatalf("failed to unmarshal result: %v", err)
			}

			if got["value"] != tt.expectValue {
				t.Errorf("value = %v, want %v", got["value"], tt.expectValue)
			}
		})
	}
}

// TestNullHandlingWithNumeric tests null handling modes with numeric strategy.
func TestNullHandlingWithNumeric(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"count": {
				"anyOf": [{"type": "integer"}, {"type": "null"}],
				"x-kfs-merge": {"strategy": "numeric", "operation": "sum"}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// A has null, B has value - numeric should use B when A is null
	a := []byte(`{"count": null}`)
	b := []byte(`{"count": 10}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	// When A is null, B's value is used
	if got["count"] != float64(10) {
		t.Errorf("count = %v, want 10", got["count"])
	}
}

// TestGlobalVsFieldLevelNullHandling tests that field-level nullHandling overrides global.
func TestGlobalVsFieldLevelNullHandling(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"x-kfs-merge": {"nullHandling": "asValue"},
		"properties": {
			"globalNull": {
				"anyOf": [{"type": "string"}, {"type": "null"}]
			},
			"fieldNull": {
				"anyOf": [{"type": "string"}, {"type": "null"}],
				"x-kfs-merge": {"nullHandling": "asAbsent"}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// A has null for both fields, B has values
	a := []byte(`{"globalNull": null, "fieldNull": null}`)
	b := []byte(`{"globalNull": "base", "fieldNull": "base"}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	// Global nullHandling is asValue - null wins
	if got["globalNull"] != nil {
		t.Errorf("globalNull = %v, want nil (global asValue)", got["globalNull"])
	}
	// Field-level nullHandling is asAbsent - B wins
	if got["fieldNull"] != "base" {
		t.Errorf("fieldNull = %v, want 'base' (field asAbsent)", got["fieldNull"])
	}
}
