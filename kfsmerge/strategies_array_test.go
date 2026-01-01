package kfsmerge

import (
	"encoding/json"
	"testing"
)

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
	b := []byte(`{"tags": ["base1", "base2"]}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	tags := got["tags"].([]any)
	// concat: B's items first, then A's items
	expected := []string{"base1", "base2", "new1", "new2"}
	if len(tags) != len(expected) {
		t.Fatalf("tags length = %d, want %d", len(tags), len(expected))
	}
	for i, v := range tags {
		if v.(string) != expected[i] {
			t.Errorf("tags[%d] = %v, want %v", i, v, expected[i])
		}
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

	a := []byte(`{"tags": ["shared", "new"]}`)
	b := []byte(`{"tags": ["base", "shared"]}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	tags := got["tags"].([]any)
	// concatUnique: B's items first, then A's unique items
	expected := []string{"base", "shared", "new"}
	if len(tags) != len(expected) {
		t.Fatalf("tags length = %d, want %d; got %v", len(tags), len(expected), tags)
	}
	for i, v := range tags {
		if v.(string) != expected[i] {
			t.Errorf("tags[%d] = %v, want %v", i, v, expected[i])
		}
	}
}

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

	a := []byte(`{"filters": [{"type": "blur", "value": 5}]}`)
	b := []byte(`{"filters": [{"type": "blur", "value": 3}, {"type": "sharpen", "value": 2}]}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	filters := got["filters"].([]any)
	// Should have 2 filters: blur (A's value) and sharpen (from B)
	if len(filters) != 2 {
		t.Fatalf("filters length = %d, want 2; got %v", len(filters), filters)
	}

	for _, f := range filters {
		filter := f.(map[string]any)
		if filter["type"] == "blur" {
			if filter["value"] != float64(5) {
				t.Errorf("blur value = %v, want 5", filter["value"])
			}
		}
	}
}

// TestMergeByDiscriminatorDefaultReplaceOnMatch ensures default behavior replaces matching items.
func TestMergeByDiscriminatorDefaultReplaceOnMatch(t *testing.T) {
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
	// With default replaceOnMatch=true, A's item should completely replace B's (no extra field)
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
			// With replaceOnMatch=true (default), extra should NOT be present (A didn't have it)
			if _, hasExtra := filter["extra"]; hasExtra {
				t.Errorf("hqdn3d.extra should not exist with replaceOnMatch=true, got %v", filter["extra"])
			}
		}
	}
}

// TestMergeByDiscriminatorDeepMergeWhenDisabled tests that replaceOnMatch=false deep merges matching items.
func TestMergeByDiscriminatorDeepMergeWhenDisabled(t *testing.T) {
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
				"x-kfs-merge": {"strategy": "mergeByDiscriminator", "discriminatorField": "type", "replaceOnMatch": false}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// B has hqdn3d with value:8 and extra:"fromB"
	// A has hqdn3d with value:12 only (no extra field)
	// With replaceOnMatch=false, deep merge should preserve B's extra field
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
