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

func TestMergeArrayConcatWithUnique(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"tags": {
				"type": "array",
				"items": {"type": "string"},
				"x-kfs-merge": {"strategy": "concat", "unique": true}
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
	// concat with unique: B's items first, then A's unique items
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

// =============================================================================
// Empty Array Handling Tests (Priority 5)
// =============================================================================

// TestConcatEmptyArrayA tests concat when A has empty array.
func TestConcatEmptyArrayA(t *testing.T) {
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

	a := []byte(`{"tags": []}`)
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
	// concat with empty A: just B's items
	expected := []string{"base1", "base2"}
	if len(tags) != len(expected) {
		t.Fatalf("tags length = %d, want %d; got %v", len(tags), len(expected), tags)
	}
	for i, v := range tags {
		if v.(string) != expected[i] {
			t.Errorf("tags[%d] = %v, want %v", i, v, expected[i])
		}
	}
}

// TestConcatEmptyArrayB tests concat when B has empty array.
func TestConcatEmptyArrayB(t *testing.T) {
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
	b := []byte(`{"tags": []}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	tags := got["tags"].([]any)
	// concat with empty B: just A's items
	expected := []string{"new1", "new2"}
	if len(tags) != len(expected) {
		t.Fatalf("tags length = %d, want %d; got %v", len(tags), len(expected), tags)
	}
	for i, v := range tags {
		if v.(string) != expected[i] {
			t.Errorf("tags[%d] = %v, want %v", i, v, expected[i])
		}
	}
}

// TestConcatBothEmptyArrays tests concat when both arrays are empty.
func TestConcatBothEmptyArrays(t *testing.T) {
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

	a := []byte(`{"tags": []}`)
	b := []byte(`{"tags": []}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	tags := got["tags"].([]any)
	// concat with both empty: empty result
	if len(tags) != 0 {
		t.Fatalf("tags length = %d, want 0; got %v", len(tags), tags)
	}
}

// TestConcatUniqueWithObjects tests that unique:true doesn't deduplicate objects.
func TestConcatUniqueWithObjects(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"items": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"name": {"type": "string"}
					}
				},
				"x-kfs-merge": {"strategy": "concat", "unique": true}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// Both have identical objects - with unique, objects should NOT be deduplicated
	a := []byte(`{"items": [{"name": "foo"}]}`)
	b := []byte(`{"items": [{"name": "foo"}]}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	items := got["items"].([]any)
	// Objects are not primitives, so unique doesn't deduplicate them
	if len(items) != 2 {
		t.Fatalf("items length = %d, want 2 (objects are not deduplicated); got %v", len(items), items)
	}
}

// TestMergeByDiscriminatorEmptyArrayA tests mergeByDiscriminator when A has empty array.
func TestMergeByDiscriminatorEmptyArrayA(t *testing.T) {
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

	a := []byte(`{"filters": []}`)
	b := []byte(`{"filters": [{"type": "blur", "value": 5}]}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	filters := got["filters"].([]any)
	// When A is empty, B's items are preserved
	if len(filters) != 1 {
		t.Fatalf("filters length = %d, want 1; got %v", len(filters), filters)
	}
	filter := filters[0].(map[string]any)
	if filter["type"] != "blur" {
		t.Errorf("filter.type = %v, want 'blur'", filter["type"])
	}
}

// TestMergeByDiscriminatorEmptyArrayB tests mergeByDiscriminator when B has empty array.
func TestMergeByDiscriminatorEmptyArrayB(t *testing.T) {
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
	b := []byte(`{"filters": []}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	filters := got["filters"].([]any)
	// When B is empty, A's items are used
	if len(filters) != 1 {
		t.Fatalf("filters length = %d, want 1; got %v", len(filters), filters)
	}
	filter := filters[0].(map[string]any)
	if filter["type"] != "blur" {
		t.Errorf("filter.type = %v, want 'blur'", filter["type"])
	}
}

// TestMergeByDiscriminatorDefaultField tests that default discriminator field is "type".
func TestMergeByDiscriminatorDefaultField(t *testing.T) {
	// No discriminatorField specified - should default to "type"
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
				"x-kfs-merge": {"strategy": "mergeByDiscriminator"}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	a := []byte(`{"filters": [{"type": "blur", "value": 10}]}`)
	b := []byte(`{"filters": [{"type": "blur", "value": 5}, {"type": "sharpen", "value": 2}]}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	filters := got["filters"].([]any)
	// Should have merged by "type" field (default)
	if len(filters) != 2 {
		t.Fatalf("filters length = %d, want 2; got %v", len(filters), filters)
	}

	// Find blur filter and check A's value was used
	for _, f := range filters {
		filter := f.(map[string]any)
		if filter["type"] == "blur" {
			if filter["value"] != float64(10) {
				t.Errorf("blur.value = %v, want 10 (A's value)", filter["value"])
			}
		}
	}
}

// TestMergeByDiscriminatorItemsWithoutDiscriminator tests handling of items without discriminator field.
func TestMergeByDiscriminatorItemsWithoutDiscriminator(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"items": {
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

	// A has item without type field
	a := []byte(`{"items": [{"value": 10}]}`)
	b := []byte(`{"items": [{"type": "blur", "value": 5}]}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	items := got["items"].([]any)
	// Both items should be in result (no match since A's item has no discriminator)
	if len(items) != 2 {
		t.Fatalf("items length = %d, want 2; got %v", len(items), items)
	}
}

// TestMergeByDiscriminatorDuplicateValues tests handling of duplicate discriminator values.
func TestMergeByDiscriminatorDuplicateValues(t *testing.T) {
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

	// B has two items with same type (duplicates)
	a := []byte(`{"filters": [{"type": "blur", "value": 10}]}`)
	b := []byte(`{"filters": [{"type": "blur", "value": 5}, {"type": "blur", "value": 3}]}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	filters := got["filters"].([]any)
	// A's blur replaces first B's blur, second B's blur is preserved
	if len(filters) != 2 {
		t.Fatalf("filters length = %d, want 2; got %v", len(filters), filters)
	}
}

// TestReplaceWithEmptyArray tests replace strategy with empty array in A.
func TestReplaceWithEmptyArray(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"x-kfs-merge": {"arrayStrategy": "replace"},
		"properties": {
			"tags": {
				"type": "array",
				"items": {"type": "string"}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// A has empty array, B has items - replace uses A's empty array
	a := []byte(`{"tags": []}`)
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
	// replace: A's empty array wins
	if len(tags) != 0 {
		t.Fatalf("tags length = %d, want 0 (replace uses A's empty array); got %v", len(tags), tags)
	}
}

// TestDefaultArrayStrategyIsReplace tests that default array strategy is replace.
func TestDefaultArrayStrategyIsReplace(t *testing.T) {
	// No x-kfs-merge on array - should use default replace strategy
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"tags": {
				"type": "array",
				"items": {"type": "string"}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	a := []byte(`{"tags": ["new1"]}`)
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
	// Default array strategy is replace: A's array wins
	if len(tags) != 1 {
		t.Fatalf("tags length = %d, want 1 (default replace strategy); got %v", len(tags), tags)
	}
	if tags[0].(string) != "new1" {
		t.Errorf("tags[0] = %v, want 'new1'", tags[0])
	}
}
