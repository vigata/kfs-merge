package kfsmerge

import (
	"encoding/json"
	"testing"
)

// =============================================================================
// Replace Strategy Tests
// =============================================================================

// TestMergeReplaceStrategyExplicit tests explicit replace strategy on objects.
func TestMergeReplaceStrategyExplicit(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"config": {
				"type": "object",
				"x-kfs-merge": {"strategy": "replace"},
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

	// A has partial config, B has full config - replace uses A entirely
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
	// replace: A's config replaces B's entirely (no port)
	if config["host"] != "production" {
		t.Errorf("config.host = %v, want 'production'", config["host"])
	}
	if _, hasPort := config["port"]; hasPort {
		t.Errorf("config.port = %v, should not exist (replace doesn't merge from B)", config["port"])
	}
}

// TestMergeReplaceWithNull tests replace when A has null.
func TestMergeReplaceWithNull(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"config": {
				"anyOf": [{"type": "object"}, {"type": "null"}],
				"x-kfs-merge": {"strategy": "replace"}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// A has null, B has object - replace should fall back to B when A is null
	a := []byte(`{"config": null}`)
	b := []byte(`{"config": {"host": "localhost"}}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	// replace: when A is nil, B is used
	config := got["config"].(map[string]any)
	if config["host"] != "localhost" {
		t.Errorf("config.host = %v, want 'localhost' (replace falls back to B when A is null)", config["host"])
	}
}

// TestMergeReplaceWhenAAbsent tests replace when A doesn't have the field.
func TestMergeReplaceWhenAAbsent(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"config": {
				"type": "object",
				"x-kfs-merge": {"strategy": "replace"},
				"properties": {
					"host": {"type": "string"}
				}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// A doesn't have config, B has it
	a := []byte(`{}`)
	b := []byte(`{"config": {"host": "localhost"}}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	// replace: when A is absent, B is preserved
	config := got["config"].(map[string]any)
	if config["host"] != "localhost" {
		t.Errorf("config.host = %v, want 'localhost' (replace preserves B when A absent)", config["host"])
	}
}

