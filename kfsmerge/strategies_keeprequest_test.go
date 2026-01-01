package kfsmerge

import (
	"encoding/json"
	"testing"
)

// =============================================================================
// keepRequest Strategy Tests
// =============================================================================

// TestMergeKeepRequestStrategy tests that keepRequest always uses A's value.
func TestMergeKeepRequestStrategy(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"override": {
				"type": "string",
				"x-kfs-merge": {"strategy": "keepRequest"}
			},
			"normal": {"type": "string"}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	a := []byte(`{"override": "from-api", "normal": "from-api"}`)
	b := []byte(`{"override": "template", "normal": "template"}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	// keepRequest: A's value always wins
	if got["override"] != "from-api" {
		t.Errorf("override = %v, want 'from-api'", got["override"])
	}
	// normal field also from A (default behavior)
	if got["normal"] != "from-api" {
		t.Errorf("normal = %v, want 'from-api'", got["normal"])
	}
}

// TestMergeKeepRequestWithNull tests keepRequest when A has null value.
func TestMergeKeepRequestWithNull(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"value": {
				"anyOf": [{"type": "string"}, {"type": "null"}],
				"x-kfs-merge": {"strategy": "keepRequest"}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// A has null, B has value - keepRequest should use A's null
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

	// keepRequest always uses A, even if A is null
	if got["value"] != nil {
		t.Errorf("value = %v, want nil (keepRequest should use A's null)", got["value"])
	}
}

// TestMergeKeepRequestWhenAAbsent tests keepRequest when A doesn't have the field.
func TestMergeKeepRequestWhenAAbsent(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"value": {
				"type": "string",
				"x-kfs-merge": {"strategy": "keepRequest"}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// A doesn't have value, B has it
	a := []byte(`{}`)
	b := []byte(`{"value": "from-base"}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	// When A is absent (field not in A at all), B's value is preserved
	// because the field is only present in B's object - deepMerge copies B's fields first
	// The keepRequest strategy is only invoked when the merge reaches a field that exists in A
	if got["value"] != "from-base" {
		t.Errorf("value = %v, want 'from-base' (field only in B, so B preserved)", got["value"])
	}
}

// TestMergeKeepRequestWithNestedObject tests keepRequest with object values.
func TestMergeKeepRequestWithNestedObject(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"config": {
				"type": "object",
				"x-kfs-merge": {"strategy": "keepRequest"},
				"properties": {
					"host": {"type": "string"},
					"port": {"type": "integer"}
				}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// A has partial config, B has full config - keepRequest uses A entirely
	a := []byte(`{"config": {"host": "production"}}`)
	b := []byte(`{"config": {"host": "localhost", "port": 8080}}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	config := got["config"].(map[string]any)
	// keepRequest: entire A's config replaces B's (no merge)
	if config["host"] != "production" {
		t.Errorf("config.host = %v, want 'production'", config["host"])
	}
	if _, hasPort := config["port"]; hasPort {
		t.Errorf("config.port = %v, should not exist (keepRequest doesn't merge from B)", config["port"])
	}
}

