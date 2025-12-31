package kfsmerge

import (
	"encoding/json"
	"testing"
)

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

func TestMergeNumericStrategies(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"sum": {
				"type": "integer",
				"x-kfs-merge": {"strategy": "sum"}
			},
			"max": {
				"type": "integer",
				"x-kfs-merge": {"strategy": "max"}
			},
			"min": {
				"type": "integer",
				"x-kfs-merge": {"strategy": "min"}
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

// TestMergeOverlay tests the overlay strategy which only applies A's explicit fields.
func TestMergeOverlay(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"config": {
				"type": "object",
				"x-kfs-merge": {"strategy": "overlay"},
				"properties": {
					"host": {"type": "string"},
					"port": {"type": "integer"},
					"timeout": {"type": "integer"},
					"debug": {"type": "boolean"}
				}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// B has all fields, A only overrides two
	b := []byte(`{"config": {"host": "localhost", "port": 5432, "timeout": 30, "debug": false}}`)
	a := []byte(`{"config": {"host": "production", "debug": true}}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	config := got["config"].(map[string]any)
	// A's fields should override B's
	if config["host"] != "production" {
		t.Errorf("config.host = %v, want 'production'", config["host"])
	}
	if config["debug"] != true {
		t.Errorf("config.debug = %v, want true", config["debug"])
	}
	// B's fields not in A should be preserved
	if config["port"] != float64(5432) {
		t.Errorf("config.port = %v, want 5432", config["port"])
	}
	if config["timeout"] != float64(30) {
		t.Errorf("config.timeout = %v, want 30", config["timeout"])
	}
}

// TestMergeOverlayNested tests overlay with nested objects.
func TestMergeOverlayNested(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"settings": {
				"type": "object",
				"x-kfs-merge": {"strategy": "overlay"},
				"properties": {
					"database": {
						"type": "object",
						"properties": {
							"host": {"type": "string"},
							"port": {"type": "integer"}
						}
					},
					"cache": {
						"type": "object",
						"properties": {
							"enabled": {"type": "boolean"},
							"ttl": {"type": "integer"}
						}
					}
				}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// B has full settings, A only updates database.host
	b := []byte(`{"settings": {
		"database": {"host": "localhost", "port": 5432},
		"cache": {"enabled": true, "ttl": 300}
	}}`)
	a := []byte(`{"settings": {
		"database": {"host": "production"}
	}}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	settings := got["settings"].(map[string]any)

	// database should be overlaid
	database := settings["database"].(map[string]any)
	if database["host"] != "production" {
		t.Errorf("database.host = %v, want 'production'", database["host"])
	}
	if database["port"] != float64(5432) {
		t.Errorf("database.port = %v, want 5432", database["port"])
	}

	// cache should be entirely from B since A doesn't have it
	cache := settings["cache"].(map[string]any)
	if cache["enabled"] != true {
		t.Errorf("cache.enabled = %v, want true", cache["enabled"])
	}
	if cache["ttl"] != float64(300) {
		t.Errorf("cache.ttl = %v, want 300", cache["ttl"])
	}
}

// TestMergeOverlayVsDeepMerge tests the difference between overlay and deepMerge.
func TestMergeOverlayVsDeepMerge(t *testing.T) {
	// With deepMerge, A's null would overwrite B's value (with default nullHandling)
	// With overlay, A only applies fields it has - it's like a PATCH operation

	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"config": {
				"type": "object",
				"x-kfs-merge": {"strategy": "overlay"},
				"properties": {
					"name": {"type": "string"},
					"value": {"type": "integer"}
				}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// A only has name, not value
	b := []byte(`{"config": {"name": "original", "value": 42}}`)
	a := []byte(`{"config": {"name": "updated"}}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	config := got["config"].(map[string]any)
	// overlay: A's name applied, B's value preserved
	if config["name"] != "updated" {
		t.Errorf("config.name = %v, want 'updated'", config["name"])
	}
	if config["value"] != float64(42) {
		t.Errorf("config.value = %v, want 42", config["value"])
	}
}
