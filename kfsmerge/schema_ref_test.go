package kfsmerge

import (
	"encoding/json"
	"testing"
)

func TestMergeWithRefAndDefs(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"$defs": {
			"ServerConfig": {
				"type": "object",
				"x-kfs-merge": {"strategy": "overlay"},
				"properties": {
					"host": {"type": "string"},
					"port": {"type": "integer"},
					"timeout": {"type": "integer"}
				}
			}
		},
		"properties": {
			"primary": {"$ref": "#/$defs/ServerConfig"},
			"secondary": {"$ref": "#/$defs/ServerConfig"}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	b := []byte(`{
		"primary": {"host": "primary.local", "port": 8080, "timeout": 30},
		"secondary": {"host": "secondary.local", "port": 9090, "timeout": 60}
	}`)
	a := []byte(`{
		"primary": {"port": 8888},
		"secondary": {"host": "new-secondary.local"}
	}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	primary := got["primary"].(map[string]any)
	// overlay: A's port applied, B's host and timeout preserved
	if primary["host"] != "primary.local" {
		t.Errorf("primary.host = %v, want 'primary.local'", primary["host"])
	}
	if primary["port"] != float64(8888) {
		t.Errorf("primary.port = %v, want 8888", primary["port"])
	}
	if primary["timeout"] != float64(30) {
		t.Errorf("primary.timeout = %v, want 30", primary["timeout"])
	}

	secondary := got["secondary"].(map[string]any)
	if secondary["host"] != "new-secondary.local" {
		t.Errorf("secondary.host = %v, want 'new-secondary.local'", secondary["host"])
	}
	if secondary["port"] != float64(9090) {
		t.Errorf("secondary.port = %v, want 9090", secondary["port"])
	}
}

func TestMergeWithNestedRefConfig(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"$defs": {
			"DatabaseConfig": {
				"type": "object",
				"x-kfs-merge": {"strategy": "deepMerge"},
				"properties": {
					"connection": {
						"type": "object",
						"properties": {
							"host": {"type": "string"},
							"port": {"type": "integer"},
							"ssl": {"type": "boolean"}
						}
					},
					"pool": {
						"type": "object",
						"properties": {
							"min": {"type": "integer"},
							"max": {"type": "integer"}
						}
					}
				}
			}
		},
		"properties": {
			"database": {"$ref": "#/$defs/DatabaseConfig"}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	b := []byte(`{
		"database": {
			"connection": {"host": "localhost", "port": 5432, "ssl": false},
			"pool": {"min": 5, "max": 20}
		}
	}`)
	a := []byte(`{
		"database": {
			"connection": {"ssl": true},
			"pool": {"max": 50}
		}
	}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	db := got["database"].(map[string]any)
	conn := db["connection"].(map[string]any)
	pool := db["pool"].(map[string]any)

	// Deep merge: A's fields override, B's fields preserved where not specified
	if conn["host"] != "localhost" {
		t.Errorf("connection.host = %v, want 'localhost'", conn["host"])
	}
	if conn["ssl"] != true {
		t.Errorf("connection.ssl = %v, want true", conn["ssl"])
	}
	if pool["min"] != float64(5) {
		t.Errorf("pool.min = %v, want 5", pool["min"])
	}
	if pool["max"] != float64(50) {
		t.Errorf("pool.max = %v, want 50", pool["max"])
	}
}

func TestMergeWithAnyOfRef(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"$defs": {
			"StringOrNull": {
				"anyOf": [
					{"type": "string"},
					{"type": "null"}
				]
			}
		},
		"properties": {
			"value": {
				"$ref": "#/$defs/StringOrNull",
				"x-kfs-merge": {"nullHandling": "preserve"}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// Test that null is preserved when A explicitly sets null
	b := []byte(`{"value": "from-base"}`)
	a := []byte(`{"value": null}`)

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

