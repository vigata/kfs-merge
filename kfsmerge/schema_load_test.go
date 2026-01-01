package kfsmerge

import (
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
			"defaultStrategy": "deepMerge",
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
