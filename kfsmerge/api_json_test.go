package kfsmerge

import "testing"

// TestMergeJSONOutput tests merge operations by comparing actual JSON output
// against expected JSON using semantic equality (ignoring field ordering and whitespace).
func TestMergeJSONOutput(t *testing.T) {
	tests := []struct {
		name     string
		schema   string
		a        string
		b        string
		expected string
	}{
		{
			name: "request overrides base fields",
			schema: `{
				"$schema": "https://json-schema.org/draft/2020-12/schema",
				"type": "object",
				"properties": {
					"name": {"type": "string"},
					"count": {"type": "integer"},
					"enabled": {"type": "boolean"}
				}
			}`,
			a:        `{"name": "from-api", "count": 10}`,
			b:        `{"name": "template", "count": 5, "enabled": true}`,
			expected: `{"count": 10, "enabled": true, "name": "from-api"}`, //knowingly changing the order
		},
		{
			name: "base fields preserved when absent in request",
			schema: `{
				"$schema": "https://json-schema.org/draft/2020-12/schema",
				"type": "object",
				"properties": {
					"host": {"type": "string"},
					"port": {"type": "integer"},
					"ssl": {"type": "boolean"}
				}
			}`,
			a:        `{"host": "production.example.com"}`,
			b:        `{"host": "localhost", "port": 5432, "ssl": false}`,
			expected: `{"host": "production.example.com", "port": 5432, "ssl": false}`,
		},
		{
			name: "deep nested merge",
			schema: `{
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
			}`,
			a:        `{"config": {"database": {"port": 5433}}}`,
			b:        `{"config": {"database": {"host": "localhost", "port": 5432}}}`,
			expected: `{"config": {"database": {"host": "localhost", "port": 5433}}}`,
		},
		{
			name: "keepBase strategy preserves base value",
			schema: `{
				"$schema": "https://json-schema.org/draft/2020-12/schema",
				"type": "object",
				"properties": {
					"immutable": {
						"type": "string",
						"x-kfs-merge": {"strategy": "keepBase"}
					},
					"mutable": {"type": "string"}
				}
			}`,
			a:        `{"immutable": "from-api", "mutable": "from-api"}`,
			b:        `{"immutable": "template", "mutable": "template"}`,
			expected: `{"immutable": "template", "mutable": "from-api"}`,
		},
		{
			name: "concat arrays",
			schema: `{
				"$schema": "https://json-schema.org/draft/2020-12/schema",
				"type": "object",
				"properties": {
					"tags": {
						"type": "array",
						"items": {"type": "string"},
						"x-kfs-merge": {"strategy": "concat"}
					}
				}
			}`,
			a:        `{"tags": ["new", "urgent"]}`,
			b:        `{"tags": ["default", "production"]}`,
			expected: `{"tags": ["default", "production", "new", "urgent"]}`,
		},
		{
			name: "concatUnique removes duplicates",
			schema: `{
				"$schema": "https://json-schema.org/draft/2020-12/schema",
				"type": "object",
				"properties": {
					"tags": {
						"type": "array",
						"items": {"type": "string"},
						"x-kfs-merge": {"strategy": "concatUnique"}
					}
				}
			}`,
			a:        `{"tags": ["production", "urgent"]}`,
			b:        `{"tags": ["default", "production"]}`,
			expected: `{"tags": ["default", "production", "urgent"]}`,
		},
		{
			name: "numeric sum strategy",
			schema: `{
				"$schema": "https://json-schema.org/draft/2020-12/schema",
				"type": "object",
				"properties": {
					"count": {
						"type": "integer",
						"x-kfs-merge": {"strategy": "sum"}
					}
				}
			}`,
			a:        `{"count": 10}`,
			b:        `{"count": 20}`,
			expected: `{"count": 30}`,
		},
		{
			name: "numeric max strategy",
			schema: `{
				"$schema": "https://json-schema.org/draft/2020-12/schema",
				"type": "object",
				"properties": {
					"limit": {
						"type": "integer",
						"x-kfs-merge": {"strategy": "max"}
					}
				}
			}`,
			a:        `{"limit": 5}`,
			b:        `{"limit": 15}`,
			expected: `{"limit": 15}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := LoadSchema([]byte(tt.schema))
			if err != nil {
				t.Fatalf("LoadSchema failed: %v", err)
			}

			result, err := s.Merge([]byte(tt.a), []byte(tt.b))
			if err != nil {
				t.Fatalf("Merge failed: %v", err)
			}

			assertJSONEqualString(t, result, tt.expected)
		})
	}
}
