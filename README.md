# kfs-flow-merge

A Go library for merging JSON instances according to JSON Schema with configurable merge rules.

## Overview

`kfs-flow-merge` takes two JSON instances (A and B) that conform to a JSON Schema, and merges them into a new instance C according to rules embedded in the schema using `x-kfs-merge` extensions.

```
┌─────────────────┐     ┌─────────────────┐
│   Instance A    │     │   Instance B    │
│  (API Request)  │     │   (Template)    │
└────────┬────────┘     └────────┬────────┘
         │                       │
         │    ┌─────────────┐    │
         └───►│   Schema    │◄───┘
              │ with merge  │
              │   rules     │
              └──────┬──────┘
                     │
                     ▼
              ┌─────────────┐
              │  Instance C │
              │  (Merged)   │
              └─────────────┘
```

**Key Features:**
- JSON Schema Draft 2020-12 validation
- Per-field merge strategy configuration via `x-kfs-merge`
- Multiple merge strategies for different data types
- Configurable null handling
- CLI tool included

## Installation

```bash
go get github.com/nbcuni/kfs-flow-merge
```

## Quick Start

```go
package main

import (
    "fmt"
    kfsmerge "github.com/nbcuni/kfs-flow-merge"
)

func main() {
    schemaJSON := []byte(`{
        "$schema": "https://json-schema.org/draft/2020-12/schema",
        "type": "object",
        "x-kfs-merge": {"defaultStrategy": "mergeRequest"},
        "properties": {
            "name": {"type": "string", "x-kfs-merge": {"strategy": "keepBase"}},
            "count": {"type": "integer"}
        }
    }`)

    schema, _ := kfsmerge.LoadSchema(schemaJSON)

    a := []byte(`{"name": "from-api", "count": 100}`)  // API request
    b := []byte(`{"name": "template", "count": 1}`)    // Template

    result, _ := schema.Merge(a, b)
    fmt.Println(string(result))
    // Output: {"count":100,"name":"template"}
    // - "name" uses keepBase, so B (base/template) wins
    // - "count" uses mergeRequest, so A (request) wins
}
```

## Merge Strategies

Configure merge behavior per-field using `x-kfs-merge`:

| Strategy | Description | Best For |
|----------|-------------|----------|
| `mergeRequest` | Request (A) wins if present, else base (B) (default) | Most fields |
| `keepBase` | Always use base's (B) value | Immutable template defaults |
| `keepRequest` | Always use request's (A) value | Required user input |
| `deepMerge` | Recursively merge objects | Nested configs |
| `replace` | Replace B's array with A's (default for arrays) | Complete replacement |
| `concat` | Append A's items to B's | Additive arrays |
| `concatUnique` | Concat and remove duplicates | Tag arrays |
| `mergeByKey` | Merge array items by a key field | Arrays of objects |
| `sum` | Add numeric values | Counters |
| `max` | Take larger value | Limits |
| `min` | Take smaller value | Thresholds |

**Note**: In `Merge(a, b)`, parameter `a` is the request/override (typically API request or user input), and parameter `b` is the base/template (typically defaults or template configuration).

### Example: Array Strategies

```json
{
  "properties": {
    "tags": {
      "type": "array",
      "x-kfs-merge": {"strategy": "concat"}
    },
    "dependencies": {
      "type": "array",
      "x-kfs-merge": {"strategy": "mergeByKey", "mergeKey": "name"}
    }
  }
}
```

## Global Configuration

Set defaults at the schema level:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "x-kfs-merge": {
    "defaultStrategy": "mergeRight",
    "arrayStrategy": "replace",
    "nullHandling": "asValue"
  }
}
```

## Null Handling

Control how explicit `null` values are handled:

| Option | Behavior |
|--------|----------|
| `asValue` | Null in A overwrites B's value (default) |
| `asAbsent` | Treat null as if the field is absent |
| `preserve` | Preserve null from A if present |

## API Reference

### Loading Schemas

```go
// From raw JSON bytes
schema, err := kfsmerge.LoadSchema(jsonBytes)

// From a file
schema, err := kfsmerge.LoadSchemaFromFile("path/to/schema.json")

// From a URL
schema, err := kfsmerge.LoadSchemaFromURL("https://example.com/schema.json")

// Auto-detect source (file, URL, or raw JSON)
schema, err := kfsmerge.LoadSchemaFromSource(source)
```

### Validation

```go
// Validate an instance against the schema
err := schema.Validate(jsonBytes)
```

### Merging

```go
// Simple merge (validates A, B, and result)
result, err := schema.Merge(instanceA, instanceB)

// Merge with options
opts := kfsmerge.MergeOptions{
    SkipValidateA:      false,  // Skip validating instance A
    SkipValidateB:      false,  // Skip validating instance B
    SkipValidateResult: false,  // Skip validating the result
}
result, err := schema.MergeWithOptions(instanceA, instanceB, opts)
```

## CLI Tool

Build and use the CLI for quick merges:

```bash
# Build
go build -o kfsmerge ./cmd/kfsmerge

# Merge two files
./kfsmerge -schema schema.json -a request.json -b template.json

# Output to file
./kfsmerge -schema schema.json -a request.json -b template.json -o result.json

# Validate only
./kfsmerge -schema schema.json -validate -a request.json

# Skip validations for faster processing
./kfsmerge -schema schema.json -a request.json -b template.json -skip-validate-result
```

### CLI Options

| Flag | Description |
|------|-------------|
| `-schema` | Path to JSON Schema file (required) |
| `-a` | Path to instance A (API request) |
| `-b` | Path to instance B (template) |
| `-o` | Output file path (default: stdout) |
| `-pretty` | Pretty-print output (default: true) |
| `-validate` | Validate inputs without merging |
| `-skip-validate-a` | Skip validation of instance A |
| `-skip-validate-b` | Skip validation of instance B |
| `-skip-validate-result` | Skip validation of merged result |

## Complete Example

### Schema (schema.json)

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "x-kfs-merge": {"defaultStrategy": "mergeRequest"},
  "properties": {
    "name": {"type": "string", "x-kfs-merge": {"strategy": "keepBase"}},
    "version": {"type": "string"},
    "config": {
      "type": "object",
      "properties": {
        "timeout": {"type": "integer"},
        "retries": {"type": "integer"}
      }
    },
    "tags": {
      "type": "array",
      "items": {"type": "string"},
      "x-kfs-merge": {"strategy": "concat"}
    },
    "dependencies": {
      "type": "array",
      "items": {"type": "object", "properties": {"name": {"type": "string"}, "version": {"type": "string"}}},
      "x-kfs-merge": {"strategy": "mergeByKey", "mergeKey": "name"}
    }
  }
}
```

### Template (template.json)

```json
{
  "name": "my-service",
  "version": "1.0.0",
  "config": {"timeout": 30, "retries": 3},
  "tags": ["production"],
  "dependencies": [{"name": "logger", "version": "2.0.0"}]
}
```

### Request (request.json)

```json
{
  "name": "custom-name",
  "version": "2.0.0",
  "config": {"timeout": 60},
  "tags": ["custom"],
  "dependencies": [{"name": "logger", "version": "3.0.0"}, {"name": "auth", "version": "1.0.0"}]
}
```

### Result

```json
{
  "name": "my-service",
  "version": "2.0.0",
  "config": {"timeout": 60, "retries": 3},
  "tags": ["production", "custom"],
  "dependencies": [
    {"name": "logger", "version": "3.0.0"},
    {"name": "auth", "version": "1.0.0"}
  ]
}
```

- **name**: Template wins (`keepLeft`)
- **version**: Request wins (`mergeRight`)
- **config**: Deep merged (timeout from request, retries from template)
- **tags**: Concatenated (`concat`)
- **dependencies**: Merged by name, logger updated, auth added (`mergeByKey`)

## License

MIT

