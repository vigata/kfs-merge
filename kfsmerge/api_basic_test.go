package kfsmerge

import (
	"encoding/json"
	"os"
	"testing"
)

func TestMergeBasic(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"count": {"type": "integer"},
			"enabled": {"type": "boolean"}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// A (API request) overrides B (template)
	a := []byte(`{"name": "from-api", "count": 10}`)
	b := []byte(`{"name": "template", "count": 5, "enabled": true}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	// A's values should win
	if got["name"] != "from-api" {
		t.Errorf("name = %v, want 'from-api'", got["name"])
	}
	if got["count"] != float64(10) {
		t.Errorf("count = %v, want 10", got["count"])
	}
	// B's value preserved when A doesn't have it
	if got["enabled"] != true {
		t.Errorf("enabled = %v, want true", got["enabled"])
	}
}

func TestMergeDeepNested(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"config": {
				"type": "object",
				"properties": {
					"database": {
						"type": "object",
						"properties": {
							"host": {"type": "string"},
							"port": {"type": "integer"}
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

	a := []byte(`{"config": {"database": {"port": 5433}}}`)
	b := []byte(`{"config": {"database": {"host": "localhost", "port": 5432}}}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	config := got["config"].(map[string]any)
	database := config["database"].(map[string]any)

	// A's port wins, B's host preserved
	if database["host"] != "localhost" {
		t.Errorf("host = %v, want 'localhost'", database["host"])
	}
	if database["port"] != float64(5433) {
		t.Errorf("port = %v, want 5433", database["port"])
	}
}

func TestMergeFromFiles(t *testing.T) {
	// Create temporary files
	schemaJSON := `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"x-kfs-merge": {"defaultStrategy": "mergeRequest"},
		"properties": {
			"name": {"type": "string"},
			"count": {"type": "integer"}
		}
	}`

	schemaFile, _ := os.CreateTemp("", "schema-*.json")
	defer os.Remove(schemaFile.Name())
	schemaFile.WriteString(schemaJSON)
	schemaFile.Close()

	aFile, _ := os.CreateTemp("", "a-*.json")
	defer os.Remove(aFile.Name())
	aFile.WriteString(`{"count": 100}`)
	aFile.Close()

	bFile, _ := os.CreateTemp("", "b-*.json")
	defer os.Remove(bFile.Name())
	bFile.WriteString(`{"name": "template", "count": 1}`)
	bFile.Close()

	// Load and merge
	s, err := LoadSchemaFromFile(schemaFile.Name())
	if err != nil {
		t.Fatalf("LoadSchemaFromFile failed: %v", err)
	}

	aData, _ := os.ReadFile(aFile.Name())
	bData, _ := os.ReadFile(bFile.Name())

	result, err := s.Merge(aData, bData)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if got["name"] != "template" {
		t.Errorf("name = %v, want 'template'", got["name"])
	}
	if got["count"] != float64(100) {
		t.Errorf("count = %v, want 100", got["count"])
	}
}

