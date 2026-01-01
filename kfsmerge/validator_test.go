package kfsmerge

import (
	"encoding/json"
	"testing"
)

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

func TestValidateAdditionalPropertiesFalse(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"count": {"type": "integer"}
		},
		"additionalProperties": false
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
		{"valid with all properties", `{"name": "test", "count": 5}`, false},
		{"valid with subset of properties", `{"name": "test"}`, false},
		{"valid empty object", `{}`, false},
		{"invalid extra property", `{"name": "test", "extra": "not allowed"}`, true},
		{"invalid unknown property only", `{"unknown": 123}`, true},
		{"invalid multiple extra properties", `{"name": "test", "foo": 1, "bar": 2}`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.Validate([]byte(tt.instance))
			if tt.wantError && err == nil {
				t.Error("expected error for additional property, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
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

func TestApplyDefaultsSchemaLevel(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"x-kfs-merge": {"applyDefaults": true},
		"properties": {
			"name": {"type": "string"},
			"timeout": {"type": "integer", "default": 30},
			"retries": {"type": "integer", "default": 3}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	tests := []struct {
		name     string
		a        string
		b        string
		expected map[string]any
	}{
		{
			name:     "defaults fill missing fields",
			a:        `{"name": "test"}`,
			b:        `{}`,
			expected: map[string]any{"name": "test", "timeout": float64(30), "retries": float64(3)},
		},
		{
			name:     "B overrides defaults",
			a:        `{"name": "test"}`,
			b:        `{"timeout": 60}`,
			expected: map[string]any{"name": "test", "timeout": float64(60), "retries": float64(3)},
		},
		{
			name:     "A overrides B overrides defaults",
			a:        `{"name": "test", "timeout": 10}`,
			b:        `{"timeout": 60}`,
			expected: map[string]any{"name": "test", "timeout": float64(10), "retries": float64(3)},
		},
		{
			name:     "all from defaults",
			a:        `{}`,
			b:        `{}`,
			expected: map[string]any{"timeout": float64(30), "retries": float64(3)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := s.MergeToValue([]byte(tt.a), []byte(tt.b))
			if err != nil {
				t.Fatalf("MergeToValue failed: %v", err)
			}

			resultMap, ok := result.(map[string]any)
			if !ok {
				t.Fatalf("result is not a map: %T", result)
			}

			for key, expectedVal := range tt.expected {
				if resultMap[key] != expectedVal {
					t.Errorf("%s = %v, want %v", key, resultMap[key], expectedVal)
				}
			}
		})
	}
}

func TestApplyDefaultsNestedObjects(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"x-kfs-merge": {"applyDefaults": true},
		"properties": {
			"config": {
				"type": "object",
				"properties": {
					"timeout": {"type": "integer", "default": 30},
					"retries": {"type": "integer", "default": 3},
					"server": {
						"type": "object",
						"properties": {
							"host": {"type": "string", "default": "localhost"},
							"port": {"type": "integer", "default": 8080}
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

	a := []byte(`{"config": {"server": {"port": 9000}}}`)
	b := []byte(`{}`)

	result, err := s.MergeToValue(a, b)
	if err != nil {
		t.Fatalf("MergeToValue failed: %v", err)
	}

	resultMap := result.(map[string]any)
	config := resultMap["config"].(map[string]any)

	if config["timeout"] != float64(30) {
		t.Errorf("config.timeout = %v, want 30", config["timeout"])
	}
	if config["retries"] != float64(3) {
		t.Errorf("config.retries = %v, want 3", config["retries"])
	}

	server := config["server"].(map[string]any)
	if server["host"] != "localhost" {
		t.Errorf("config.server.host = %v, want localhost", server["host"])
	}
	if server["port"] != float64(9000) {
		t.Errorf("config.server.port = %v, want 9000 (A's value)", server["port"])
	}
}

func TestApplyDefaultsMergeOptionsOverride(t *testing.T) {
	// Schema has applyDefaults: false
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"timeout": {"type": "integer", "default": 30}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	a := []byte(`{}`)
	b := []byte(`{}`)

	// Without override, defaults should NOT be applied
	result, err := s.MergeToValue(a, b)
	if err != nil {
		t.Fatalf("MergeToValue failed: %v", err)
	}
	resultMap := result.(map[string]any)
	if _, exists := resultMap["timeout"]; exists {
		t.Error("timeout should not exist without applyDefaults")
	}

	// With override, defaults SHOULD be applied
	applyDefaults := true
	opts := MergeOptions{ApplyDefaults: &applyDefaults}
	result, err = s.MergeToValueWithOptions(a, b, opts)
	if err != nil {
		t.Fatalf("MergeToValueWithOptions failed: %v", err)
	}
	resultMap = result.(map[string]any)
	if resultMap["timeout"] != float64(30) {
		t.Errorf("timeout = %v, want 30", resultMap["timeout"])
	}
}

func TestApplyDefaultsWithObjectLevelDefault(t *testing.T) {
	// Test case where object has both object-level default and property defaults
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"x-kfs-merge": {"applyDefaults": true},
		"properties": {
			"config": {
				"type": "object",
				"default": {"timeout": 60, "enabled": true},
				"properties": {
					"timeout": {"type": "integer", "default": 30},
					"retries": {"type": "integer", "default": 3}
				}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// Verify extracted defaults: leaf defaults should override object default
	defaults := s.Defaults()
	if defaults == nil {
		t.Fatal("Defaults() returned nil")
	}

	config := defaults["config"].(map[string]any)
	// timeout: leaf default (30) should override object default (60)
	if config["timeout"] != float64(30) {
		t.Errorf("default config.timeout = %v, want 30 (leaf wins)", config["timeout"])
	}
	// retries: only in leaf defaults
	if config["retries"] != float64(3) {
		t.Errorf("default config.retries = %v, want 3", config["retries"])
	}
	// enabled: only in object default, should be preserved
	if config["enabled"] != true {
		t.Errorf("default config.enabled = %v, want true (from object default)", config["enabled"])
	}
}

func TestApplyDefaultsWithRef(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"x-kfs-merge": {"applyDefaults": true},
		"$defs": {
			"Timeout": {
				"type": "integer",
				"default": 30
			}
		},
		"properties": {
			"requestTimeout": {"$ref": "#/$defs/Timeout"},
			"responseTimeout": {"$ref": "#/$defs/Timeout"}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	a := []byte(`{"requestTimeout": 10}`)
	b := []byte(`{}`)

	result, err := s.MergeToValue(a, b)
	if err != nil {
		t.Fatalf("MergeToValue failed: %v", err)
	}

	resultMap := result.(map[string]any)
	if resultMap["requestTimeout"] != float64(10) {
		t.Errorf("requestTimeout = %v, want 10 (from A)", resultMap["requestTimeout"])
	}
	if resultMap["responseTimeout"] != float64(30) {
		t.Errorf("responseTimeout = %v, want 30 (from default)", resultMap["responseTimeout"])
	}
}
