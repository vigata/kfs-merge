package kfsmerge

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

// jsonEqual compares two JSON byte slices for semantic equality.
// Returns true if they represent the same JSON value, ignoring
// field ordering and whitespace differences.
func jsonEqual(a, b []byte) (bool, error) {
	var aVal, bVal any
	if err := json.Unmarshal(a, &aVal); err != nil {
		return false, fmt.Errorf("failed to unmarshal first JSON: %w", err)
	}
	if err := json.Unmarshal(b, &bVal); err != nil {
		return false, fmt.Errorf("failed to unmarshal second JSON: %w", err)
	}
	return reflect.DeepEqual(aVal, bVal), nil
}

// assertJSONEqual fails the test if actual and expected are not semantically equal.
// Provides a clear diff on failure by showing both in canonical form.
func assertJSONEqual(t *testing.T, actual, expected []byte) {
	t.Helper()

	equal, err := jsonEqual(actual, expected)
	if err != nil {
		t.Fatalf("JSON comparison error: %v", err)
	}

	if !equal {
		// Re-marshal to canonical form for clear diff
		var aVal, eVal any
		json.Unmarshal(actual, &aVal)
		json.Unmarshal(expected, &eVal)

		canonicalActual, _ := json.MarshalIndent(aVal, "", "  ")
		canonicalExpected, _ := json.MarshalIndent(eVal, "", "  ")

		t.Errorf("JSON mismatch:\ngot:\n%s\n\nwant:\n%s", canonicalActual, canonicalExpected)
	}
}

// assertJSONEqualString is a convenience wrapper that accepts string arguments.
func assertJSONEqualString(t *testing.T, actual []byte, expected string) {
	t.Helper()
	assertJSONEqual(t, actual, []byte(expected))
}

// TestJsonEqualFunction tests the jsonEqual helper function.
func TestJsonEqualFunction(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected bool
	}{
		{
			name:     "identical JSON",
			a:        `{"name": "test", "count": 42}`,
			b:        `{"name": "test", "count": 42}`,
			expected: true,
		},
		{
			name:     "different field ordering same content",
			a:        `{"count": 42, "name": "test"}`,
			b:        `{"name": "test", "count": 42}`,
			expected: true,
		},
		{
			name:     "different whitespace same content",
			a:        `{"name":"test","count":42}`,
			b:        `{ "name" : "test" , "count" : 42 }`,
			expected: true,
		},
		{
			name:     "nested objects different ordering",
			a:        `{"outer": {"b": 2, "a": 1}}`,
			b:        `{"outer": {"a": 1, "b": 2}}`,
			expected: true,
		},
		{
			name:     "different values",
			a:        `{"name": "foo"}`,
			b:        `{"name": "bar"}`,
			expected: false,
		},
		{
			name:     "different types",
			a:        `{"count": 42}`,
			b:        `{"count": "42"}`,
			expected: false,
		},
		{
			name:     "missing field",
			a:        `{"name": "test", "count": 42}`,
			b:        `{"name": "test"}`,
			expected: false,
		},
		{
			name:     "array ordering matters",
			a:        `{"items": [1, 2, 3]}`,
			b:        `{"items": [3, 2, 1]}`,
			expected: false,
		},
		{
			name:     "null vs missing field",
			a:        `{"name": null}`,
			b:        `{}`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			equal, err := jsonEqual([]byte(tt.a), []byte(tt.b))
			if err != nil {
				t.Fatalf("jsonEqual error: %v", err)
			}
			if equal != tt.expected {
				t.Errorf("jsonEqual(%s, %s) = %v, want %v", tt.a, tt.b, equal, tt.expected)
			}
		})
	}
}
