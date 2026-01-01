package kfsmerge

import (
	"encoding/json"
	"testing"
)

// =============================================================================
// keepBase Strategy Tests
// =============================================================================

func TestMergeKeepBaseStrategy(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"immutable": {
				"type": "string",
				"x-kfs-merge": {"strategy": "keepBase"}
			},
			"mutable": {"type": "string"}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	a := []byte(`{"immutable": "from-api", "mutable": "from-api"}`)
	b := []byte(`{"immutable": "template", "mutable": "template"}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	// keepBase: B's value preserved
	if got["immutable"] != "template" {
		t.Errorf("immutable = %v, want 'template'", got["immutable"])
	}
	// default: A's value wins
	if got["mutable"] != "from-api" {
		t.Errorf("mutable = %v, want 'from-api'", got["mutable"])
	}
}

// TestMergeKeepBaseWithNullInA tests keepBase when A has null value.
func TestMergeKeepBaseWithNullInA(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"value": {
				"anyOf": [{"type": "string"}, {"type": "null"}],
				"x-kfs-merge": {"strategy": "keepBase"}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// A has null, B has value - keepBase should always use B
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

	// keepBase always uses B, ignoring A's null
	if got["value"] != "from-base" {
		t.Errorf("value = %v, want 'from-base' (keepBase should use B)", got["value"])
	}
}

// TestMergeKeepBaseWhenBIsNull tests keepBase when B has null value.
func TestMergeKeepBaseWhenBIsNull(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"value": {
				"anyOf": [{"type": "string"}, {"type": "null"}],
				"x-kfs-merge": {"strategy": "keepBase"}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// A has value, B has null - keepBase should use B's null
	a := []byte(`{"value": "from-api"}`)
	b := []byte(`{"value": null}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	// keepBase always uses B, even if B is null
	if got["value"] != nil {
		t.Errorf("value = %v, want nil (keepBase should use B's null)", got["value"])
	}
}

// TestMergeKeepBaseWithNestedObject tests keepBase with nested objects.
func TestMergeKeepBaseWithNestedObject(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"config": {
				"type": "object",
				"x-kfs-merge": {"strategy": "keepBase"},
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

	// A has different config, B has base config - keepBase uses B entirely
	a := []byte(`{"config": {"host": "production", "port": 9000}}`)
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
	// keepBase: entire B's config used
	if config["host"] != "localhost" {
		t.Errorf("config.host = %v, want 'localhost'", config["host"])
	}
	if config["port"] != float64(8080) {
		t.Errorf("config.port = %v, want 8080", config["port"])
	}
}

