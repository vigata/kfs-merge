package kfsmerge

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

func TestLoadSchema(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"count": {"type": "integer"}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}
	if s == nil {
		t.Fatal("LoadSchema returned nil")
	}
}

func TestLoadSchemaWithMergeExtensions(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"x-kfs-merge": {
			"defaultStrategy": "mergeRequest",
			"arrayStrategy": "concat"
		},
		"properties": {
			"name": {
				"type": "string",
				"x-kfs-merge": {"strategy": "keepBase"}
			},
			"tags": {
				"type": "array",
				"items": {"type": "string"},
				"x-kfs-merge": {"strategy": "concat"}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}
	if s == nil {
		t.Fatal("LoadSchema returned nil")
	}
}

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

func TestMergeKeepBaseStrategy(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"name": {
				"type": "string",
				"x-kfs-merge": {"strategy": "keepBase"}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	a := []byte(`{"name": "from-api"}`)
	b := []byte(`{"name": "template"}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	// keepBase means B wins
	if got["name"] != "template" {
		t.Errorf("name = %v, want 'template'", got["name"])
	}
}

func TestMergeArrayConcat(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"tags": {
				"type": "array",
				"items": {"type": "string"},
				"x-kfs-merge": {"strategy": "concat"}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	a := []byte(`{"tags": ["new1", "new2"]}`)
	b := []byte(`{"tags": ["existing"]}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	tags := got["tags"].([]any)
	if len(tags) != 3 {
		t.Errorf("tags length = %d, want 3", len(tags))
	}
	// B first, then A
	if tags[0] != "existing" || tags[1] != "new1" || tags[2] != "new2" {
		t.Errorf("tags = %v, want [existing, new1, new2]", tags)
	}
}

func TestMergeArrayConcatUnique(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"tags": {
				"type": "array",
				"items": {"type": "string"},
				"x-kfs-merge": {"strategy": "concatUnique"}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	a := []byte(`{"tags": ["a", "b", "c"]}`)
	b := []byte(`{"tags": ["b", "c", "d"]}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	tags := got["tags"].([]any)
	// B first, then A unique items: b, c, d, a
	if len(tags) != 4 {
		t.Errorf("tags length = %d, want 4, got %v", len(tags), tags)
	}
}

func TestMergeByKey(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"items": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"id": {"type": "string"},
						"value": {"type": "integer"}
					}
				},
				"x-kfs-merge": {"strategy": "mergeByKey", "mergeKey": "id"}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	a := []byte(`{"items": [
		{"id": "a", "value": 100},
		{"id": "c", "value": 300}
	]}`)
	b := []byte(`{"items": [
		{"id": "a", "value": 1},
		{"id": "b", "value": 2}
	]}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	items := got["items"].([]any)
	// A's "a" merged with B's "a" (A wins), A's "c" added, B's "b" added
	if len(items) != 3 {
		t.Errorf("items length = %d, want 3", len(items))
	}

	// Find item with id "a" and check value
	for _, item := range items {
		obj := item.(map[string]any)
		if obj["id"] == "a" {
			if obj["value"] != float64(100) {
				t.Errorf("item 'a' value = %v, want 100", obj["value"])
			}
		}
	}
}

func TestMergeNumericStrategies(t *testing.T) {
	tests := []struct {
		name     string
		strategy string
		aVal     int
		bVal     int
		expected float64
	}{
		{"sum", "sum", 10, 5, 15},
		{"max - a wins", "max", 10, 5, 10},
		{"max - b wins", "max", 3, 8, 8},
		{"min - a wins", "min", 3, 8, 3},
		{"min - b wins", "min", 10, 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schemaJSON := []byte(`{
				"$schema": "https://json-schema.org/draft/2020-12/schema",
				"type": "object",
				"properties": {
					"count": {
						"type": "integer",
						"x-kfs-merge": {"strategy": "` + tt.strategy + `"}
					}
				}
			}`)

			s, err := LoadSchema(schemaJSON)
			if err != nil {
				t.Fatalf("LoadSchema failed: %v", err)
			}

			a := []byte(fmt.Sprintf(`{"count": %d}`, tt.aVal))
			b := []byte(fmt.Sprintf(`{"count": %d}`, tt.bVal))

			result, err := s.Merge(a, b)
			if err != nil {
				t.Fatalf("Merge failed: %v", err)
			}

			var got map[string]any
			if err := json.Unmarshal(result, &got); err != nil {
				t.Fatalf("failed to unmarshal result: %v", err)
			}

			if got["count"] != tt.expected {
				t.Errorf("count = %v, want %v", got["count"], tt.expected)
			}
		})
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

func TestMergeNullHandlingAsValue(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"x-kfs-merge": {"nullHandling": "asValue"},
		"properties": {
			"name": {"type": ["string", "null"]}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// A has explicit null, should overwrite B's value
	a := []byte(`{"name": null}`)
	b := []byte(`{"name": "template"}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	// nullAsValue means A's null overwrites B
	if got["name"] != nil {
		t.Errorf("name = %v, want null", got["name"])
	}
}

func TestMergeNullHandlingAsAbsent(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"x-kfs-merge": {"nullHandling": "asAbsent"},
		"properties": {
			"name": {"type": ["string", "null"]}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// A has explicit null, but with asAbsent it should use B's value
	a := []byte(`{"name": null}`)
	b := []byte(`{"name": "template"}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	// nullAsAbsent means A's null is treated as absent, so B wins
	if got["name"] != "template" {
		t.Errorf("name = %v, want 'template'", got["name"])
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

func TestLoadSchemaFromFile(t *testing.T) {
	// Create a temporary schema file
	schemaJSON := `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		}
	}`

	tmpFile, err := os.CreateTemp("", "schema-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(schemaJSON); err != nil {
		t.Fatalf("failed to write schema: %v", err)
	}
	tmpFile.Close()

	s, err := LoadSchemaFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadSchemaFromFile failed: %v", err)
	}

	// Test that the schema works
	err = s.Validate([]byte(`{"name": "test"}`))
	if err != nil {
		t.Errorf("validation failed: %v", err)
	}
}

func TestLoadSchemaFromSource(t *testing.T) {
	tests := []struct {
		name   string
		source string
		valid  bool
	}{
		{
			name: "raw JSON",
			source: `{
				"$schema": "https://json-schema.org/draft/2020-12/schema",
				"type": "object"
			}`,
			valid: true,
		},
		{
			name:   "non-existent file",
			source: "/nonexistent/path/schema.json",
			valid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := LoadSchemaFromSource(tt.source)
			if tt.valid {
				if err != nil {
					t.Errorf("LoadSchemaFromSource failed: %v", err)
				}
				if s == nil {
					t.Error("expected non-nil schema")
				}
			} else {
				if err == nil {
					t.Error("expected error, got nil")
				}
			}
		})
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

func TestMergeWithRefAndDefs(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"$defs": {
			"Config": {
				"type": "object",
				"x-kfs-merge": {"strategy": "keepBase"},
				"properties": {
					"name": {"type": "string"},
					"value": {"type": "integer"}
				}
			}
		},
		"properties": {
			"config": {"$ref": "#/$defs/Config"}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// A tries to override, but keepBase means B wins
	a := []byte(`{"config": {"name": "from-api", "value": 100}}`)
	b := []byte(`{"config": {"name": "template", "value": 1}}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	config := got["config"].(map[string]any)
	// keepBase means B wins
	if config["name"] != "template" {
		t.Errorf("config.name = %v, want 'template'", config["name"])
	}
	if config["value"] != float64(1) {
		t.Errorf("config.value = %v, want 1", config["value"])
	}
}

func TestMergeWithNestedRefConfig(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"$defs": {
			"DatabaseConfig": {
				"type": "object",
				"properties": {
					"host": {"type": "string"},
					"port": {
						"type": "integer",
						"x-kfs-merge": {"strategy": "keepBase"}
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

	// A overrides host (mergeRequest default), but port uses keepBase
	a := []byte(`{"database": {"host": "prod-server", "port": 5433}}`)
	b := []byte(`{"database": {"host": "localhost", "port": 5432}}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	database := got["database"].(map[string]any)
	// host uses default mergeRequest, A wins
	if database["host"] != "prod-server" {
		t.Errorf("database.host = %v, want 'prod-server'", database["host"])
	}
	// port uses keepBase from $defs, B wins
	if database["port"] != float64(5432) {
		t.Errorf("database.port = %v, want 5432", database["port"])
	}
}

func TestMergeWithAnyOfRef(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"$defs": {
			"OptionalConfig": {
				"type": "object",
				"x-kfs-merge": {"strategy": "keepBase"},
				"properties": {
					"enabled": {"type": "boolean"}
				}
			}
		},
		"properties": {
			"settings": {
				"anyOf": [
					{"$ref": "#/$defs/OptionalConfig"},
					{"type": "null"}
				]
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// A tries to override, but anyOf with $ref should use keepBase from def
	a := []byte(`{"settings": {"enabled": true}}`)
	b := []byte(`{"settings": {"enabled": false}}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	settings := got["settings"].(map[string]any)
	// keepBase means B wins
	if settings["enabled"] != false {
		t.Errorf("settings.enabled = %v, want false", settings["enabled"])
	}
}

// TestKfsMediaSchemaLoad tests that the production schema can be loaded.
func TestKfsMediaSchemaLoad(t *testing.T) {
	s, err := LoadSchemaFromFile("examples/kfs_media_schema.json")
	if err != nil {
		t.Fatalf("LoadSchemaFromFile failed: %v", err)
	}
	if s == nil {
		t.Fatal("LoadSchemaFromFile returned nil")
	}

	// Verify some expected $defs were parsed
	// The schema should have recognized the $defs
	t.Log("Successfully loaded kfs_media_schema.json")
}

// TestKfsMediaSchemaMergeBackend tests merging the backend field which uses anyOf with $ref.
func TestKfsMediaSchemaMergeBackend(t *testing.T) {
	s, err := LoadSchemaFromFile("examples/kfs_media_schema.json")
	if err != nil {
		t.Fatalf("LoadSchemaFromFile failed: %v", err)
	}

	// Template (B) has default backend settings
	template := []byte(`{
		"source_file_path": "s3://bucket/source.mp4",
		"intermediate_path": "s3://bucket/intermediate/",
		"temporary_storage_path": "s3://bucket/temp/",
		"profile_configuration": {"profile_name": "test", "operation_mode": "encode"},
		"profile_caedl_pars": {},
		"generic_caedl_pars": {},
		"backend": {
			"selected": "json_runner",
			"options": {
				"json_runner": {
					"json_path": "tasks.json",
					"report_level": "medium_high"
				}
			}
		}
	}`)

	// Request (A) overrides just the selected backend
	request := []byte(`{
		"source_file_path": "s3://bucket/source.mp4",
		"intermediate_path": "s3://bucket/intermediate/",
		"temporary_storage_path": "s3://bucket/temp/",
		"profile_configuration": {"profile_name": "test", "operation_mode": "encode"},
		"profile_caedl_pars": {},
		"generic_caedl_pars": {},
		"backend": {
			"selected": "hybrik"
		}
	}`)

	opts := MergeOptions{SkipValidateResult: true}
	result, err := s.MergeWithOptions(request, template, opts)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	backend := got["backend"].(map[string]any)
	// A's selected should override B's
	if backend["selected"] != "hybrik" {
		t.Errorf("backend.selected = %v, want 'hybrik'", backend["selected"])
	}
	// B's options should be preserved (deep merge)
	options, hasOptions := backend["options"].(map[string]any)
	if !hasOptions {
		t.Error("backend.options not preserved from template")
	} else {
		jsonRunner, hasJsonRunner := options["json_runner"].(map[string]any)
		if !hasJsonRunner {
			t.Error("backend.options.json_runner not preserved from template")
		} else {
			if jsonRunner["json_path"] != "tasks.json" {
				t.Errorf("json_runner.json_path = %v, want 'tasks.json'", jsonRunner["json_path"])
			}
		}
	}
}

// TestKfsMediaSchemaMergeRunningOptions tests merging nested running_options.
func TestKfsMediaSchemaMergeRunningOptions(t *testing.T) {
	s, err := LoadSchemaFromFile("examples/kfs_media_schema.json")
	if err != nil {
		t.Fatalf("LoadSchemaFromFile failed: %v", err)
	}

	// Template with default running options
	template := []byte(`{
		"source_file_path": "s3://bucket/source.mp4",
		"intermediate_path": "s3://bucket/intermediate/",
		"temporary_storage_path": "s3://bucket/temp/",
		"profile_configuration": {"profile_name": "test", "operation_mode": "encode"},
		"profile_caedl_pars": {},
		"generic_caedl_pars": {},
		"running_options": {
			"threads_per_task": 6,
			"job_timeout_seconds": 86400,
			"logging_options": {
				"log_level": "INFO"
			}
		}
	}`)

	// Request overrides threads and log_level
	request := []byte(`{
		"source_file_path": "s3://bucket/source.mp4",
		"intermediate_path": "s3://bucket/intermediate/",
		"temporary_storage_path": "s3://bucket/temp/",
		"profile_configuration": {"profile_name": "test", "operation_mode": "encode"},
		"profile_caedl_pars": {},
		"generic_caedl_pars": {},
		"running_options": {
			"threads_per_task": 12,
			"logging_options": {
				"log_level": "DEBUG"
			}
		}
	}`)

	opts := MergeOptions{SkipValidateResult: true}
	result, err := s.MergeWithOptions(request, template, opts)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	runningOpts := got["running_options"].(map[string]any)
	// A's threads_per_task should override B's
	if runningOpts["threads_per_task"] != float64(12) {
		t.Errorf("threads_per_task = %v, want 12", runningOpts["threads_per_task"])
	}
	// B's job_timeout_seconds should be preserved
	if runningOpts["job_timeout_seconds"] != float64(86400) {
		t.Errorf("job_timeout_seconds = %v, want 86400", runningOpts["job_timeout_seconds"])
	}

	loggingOpts := runningOpts["logging_options"].(map[string]any)
	// A's log_level should override B's
	if loggingOpts["log_level"] != "DEBUG" {
		t.Errorf("log_level = %v, want 'DEBUG'", loggingOpts["log_level"])
	}
}

// TestMergeByDiscriminator tests merging arrays of discriminated union objects.
func TestMergeByDiscriminator(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"filters": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"type": {"type": "string"},
						"value": {"type": "integer"}
					}
				},
				"x-kfs-merge": {"strategy": "mergeByDiscriminator", "discriminatorField": "type"}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// B has hqdn3d with luma:8 and unsharp with amount:1
	// A has hqdn3d with luma:12 (override) and no unsharp
	b := []byte(`{"filters": [
		{"type": "hqdn3d", "value": 8},
		{"type": "unsharp", "value": 1}
	]}`)
	a := []byte(`{"filters": [
		{"type": "hqdn3d", "value": 12}
	]}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	filters := got["filters"].([]any)
	// Should have 2 items: hqdn3d (merged) and unsharp (preserved from B)
	if len(filters) != 2 {
		t.Fatalf("filters length = %d, want 2; got %v", len(filters), filters)
	}

	// Find each filter by type
	hqdn3dFound := false
	unsharpFound := false
	for _, f := range filters {
		filter := f.(map[string]any)
		switch filter["type"] {
		case "hqdn3d":
			hqdn3dFound = true
			// A's value should override B's
			if filter["value"] != float64(12) {
				t.Errorf("hqdn3d.value = %v, want 12", filter["value"])
			}
		case "unsharp":
			unsharpFound = true
			// B's value should be preserved
			if filter["value"] != float64(1) {
				t.Errorf("unsharp.value = %v, want 1", filter["value"])
			}
		}
	}
	if !hqdn3dFound {
		t.Error("hqdn3d filter not found in result")
	}
	if !unsharpFound {
		t.Error("unsharp filter not found in result")
	}
}

// TestMergeByDiscriminatorNewType tests adding a new type via A.
func TestMergeByDiscriminatorNewType(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"filters": {
				"type": "array",
				"items": {"type": "object"},
				"x-kfs-merge": {"strategy": "mergeByDiscriminator", "discriminatorField": "type"}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// B has one filter, A adds a new type
	b := []byte(`{"filters": [{"type": "blur", "radius": 5}]}`)
	a := []byte(`{"filters": [{"type": "sharpen", "amount": 1.5}]}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	filters := got["filters"].([]any)
	// Should have both filters
	if len(filters) != 2 {
		t.Fatalf("filters length = %d, want 2", len(filters))
	}
}

// TestMergeByDiscriminatorReplaceOnMatch tests that replaceOnMatch replaces instead of deep merging.
func TestMergeByDiscriminatorReplaceOnMatch(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"filters": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"type": {"type": "string"},
						"value": {"type": "integer"},
						"extra": {"type": "string"}
					}
				},
				"x-kfs-merge": {"strategy": "mergeByDiscriminator", "discriminatorField": "type", "replaceOnMatch": true}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// B has hqdn3d with value:8 and extra:"fromB"
	// A has hqdn3d with value:12 only (no extra field)
	// With replaceOnMatch=true, A's item should completely replace B's (no extra field)
	b := []byte(`{"filters": [
		{"type": "hqdn3d", "value": 8, "extra": "fromB"},
		{"type": "unsharp", "value": 1}
	]}`)
	a := []byte(`{"filters": [
		{"type": "hqdn3d", "value": 12}
	]}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	filters := got["filters"].([]any)
	// Should have 2 items: hqdn3d (replaced) and unsharp (preserved from B)
	if len(filters) != 2 {
		t.Fatalf("filters length = %d, want 2; got %v", len(filters), filters)
	}

	// Find hqdn3d filter
	for _, f := range filters {
		filter := f.(map[string]any)
		if filter["type"] == "hqdn3d" {
			// A's value should be used
			if filter["value"] != float64(12) {
				t.Errorf("hqdn3d.value = %v, want 12", filter["value"])
			}
			// With replaceOnMatch=true, extra should NOT be present (A didn't have it)
			if _, hasExtra := filter["extra"]; hasExtra {
				t.Errorf("hqdn3d.extra should not exist with replaceOnMatch=true, got %v", filter["extra"])
			}
		}
	}
}

// TestMergeByKeyReplaceOnMatch tests that replaceOnMatch works with mergeByKey strategy.
func TestMergeByKeyReplaceOnMatch(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"items": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"id": {"type": "string"},
						"name": {"type": "string"},
						"value": {"type": "integer"}
					}
				},
				"x-kfs-merge": {"strategy": "mergeByKey", "mergeKey": "id", "replaceOnMatch": true}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// B has item1 with name and value
	// A has item1 with only value (no name)
	// With replaceOnMatch=true, A's item should completely replace B's (no name field)
	b := []byte(`{"items": [
		{"id": "item1", "name": "Original", "value": 100},
		{"id": "item2", "name": "Second", "value": 200}
	]}`)
	a := []byte(`{"items": [
		{"id": "item1", "value": 999}
	]}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	items := got["items"].([]any)
	// Should have 2 items: item1 (replaced) and item2 (preserved from B)
	if len(items) != 2 {
		t.Fatalf("items length = %d, want 2; got %v", len(items), items)
	}

	// Find item1
	for _, i := range items {
		item := i.(map[string]any)
		if item["id"] == "item1" {
			// A's value should be used
			if item["value"] != float64(999) {
				t.Errorf("item1.value = %v, want 999", item["value"])
			}
			// With replaceOnMatch=true, name should NOT be present (A didn't have it)
			if _, hasName := item["name"]; hasName {
				t.Errorf("item1.name should not exist with replaceOnMatch=true, got %v", item["name"])
			}
		}
	}
}

// TestMergeByDiscriminatorDeepMergeDefault tests that without replaceOnMatch, deep merge is used.
func TestMergeByDiscriminatorDeepMergeDefault(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"filters": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"type": {"type": "string"},
						"value": {"type": "integer"},
						"extra": {"type": "string"}
					}
				},
				"x-kfs-merge": {"strategy": "mergeByDiscriminator", "discriminatorField": "type"}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// B has hqdn3d with value:8 and extra:"fromB"
	// A has hqdn3d with value:12 only (no extra field)
	// Without replaceOnMatch (default false), deep merge should preserve B's extra field
	b := []byte(`{"filters": [
		{"type": "hqdn3d", "value": 8, "extra": "fromB"}
	]}`)
	a := []byte(`{"filters": [
		{"type": "hqdn3d", "value": 12}
	]}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	filters := got["filters"].([]any)
	if len(filters) != 1 {
		t.Fatalf("filters length = %d, want 1; got %v", len(filters), filters)
	}

	filter := filters[0].(map[string]any)
	// A's value should override B's
	if filter["value"] != float64(12) {
		t.Errorf("hqdn3d.value = %v, want 12", filter["value"])
	}
	// Without replaceOnMatch, extra should be preserved from B
	if filter["extra"] != "fromB" {
		t.Errorf("hqdn3d.extra = %v, want 'fromB' (should be preserved with deep merge)", filter["extra"])
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
