# KFS-Flow-Merge: Comprehensive Merge Strategies Guide

## Table of Contents

- [Introduction](#introduction)
- [Strategy Reference](#strategy-reference)
  - [1. mergeRequest (Default)](#1-mergerequest-default)
  - [2. keepBase](#2-keepbase)
  - [3. keepRequest](#3-keeprequest)
  - [4. deepMerge](#4-deepmerge)
  - [5. replace](#5-replace)
  - [6. concat](#6-concat)
  - [7. concatUnique](#7-concatunique)
  - [8. mergeByKey](#8-mergebykey)
  - [9. mergeByDiscriminator](#9-mergebydiscriminator)
  - [10. overlay](#10-overlay)
  - [11. sum](#11-sum)
  - [12. max](#12-max)
  - [13. min](#13-min)
- [Best Practices](#best-practices)
- [Common Pitfalls to Avoid](#common-pitfalls-to-avoid)
- [Complete Real-World Example](#complete-real-world-example)
- [Summary](#summary)

---

## Introduction

The kfs-flow-merge library provides 13 distinct merge strategies to control how JSON instances are combined. Each strategy is configured using the `x-kfs-merge` extension in your JSON Schema, allowing fine-grained control over merge behavior at every level of your data structure.

### Core Concept

When merging two JSON instances using `Merge(a, b)`:
- **Instance A** (first parameter): The request/override instance (typically an API request or user input)
- **Instance B** (second parameter): The base/template instance (typically defaults or template configuration)
- **Result C**: The merged output

By default, A takes precedence over B (request overrides base), but each strategy modifies this behavior differently.

### Configuration Syntax

```json
{
  "x-kfs-merge": {
    "strategy": "strategyName",
    "mergeKey": "keyField",           // For mergeByKey strategy
    "discriminatorField": "type",     // For mergeByDiscriminator strategy
    "replaceOnMatch": false,          // For mergeByKey/mergeByDiscriminator: replace instead of deep merge
    "defaultStrategy": "mergeRequest", // Default for all fields
    "arrayStrategy": "replace",       // Default for arrays
    "nullHandling": "asAbsent"        // How to treat null values
  }
}
```

---

## Strategy Reference

### 1. mergeRequest (Default)

**Description**: The request's (A) value wins if present and non-null, otherwise the base's (B) value is used. For objects, performs recursive deep merge.

**When to Use**:
- Default behavior for most fields
- API requests overriding template defaults
- Standard configuration merging

**Example**:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "x-kfs-merge": {"defaultStrategy": "mergeRequest"},
  "properties": {
    "job_timeout_seconds": {"type": "integer"},
    "max_attempts": {"type": "integer"}
  }
}
```

**Input A** (API Request):
```json
{
  "job_timeout_seconds": 3600
}
```

**Input B** (Base/Template):
```json
{
  "job_timeout_seconds": 86400,
  "max_attempts": 10
}
```

**Result**:
```json
{
  "job_timeout_seconds": 3600,
  "max_attempts": 10
}
```

**Explanation**: A's `job_timeout_seconds` (3600) overrides B's value. A doesn't specify `max_attempts`, so B's value (10) is preserved.

---

### 2. keepBase

**Description**: Always uses the base's (B) value, ignoring the request (A) completely. The base/template always wins.

**When to Use**:
- Enforcing immutable template defaults
- System-controlled values that users cannot override
- Security-critical settings

**Example**:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "retry_mode": {
      "type": "string",
      "x-kfs-merge": {"strategy": "keepBase"}
    },
    "max_attempts": {"type": "integer"}
  }
}
```

**Input A** (API Request):
```json
{
  "retry_mode": "aggressive",
  "max_attempts": 20
}
```

**Input B** (Base/Template):
```json
{
  "retry_mode": "standard",
  "max_attempts": 10
}
```

**Result**:
```json
{
  "retry_mode": "standard",
  "max_attempts": 20
}
```

**Explanation**: `retry_mode` uses `keepBase`, so B's "standard" is preserved despite A specifying "aggressive". `max_attempts` uses default `mergeRequest`, so A's 20 wins.

---

### 3. keepRequest

**Description**: Always uses the request's (A) value, ignoring the base (B) completely. The request always wins.

**When to Use**:
- User must explicitly provide values
- No fallback to template defaults desired
- Complete user control over specific fields

**Example**:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "output_file_pattern": {
      "type": "string",
      "x-kfs-merge": {"strategy": "keepRequest"}
    }
  }
}
```

**Input A** (API Request):
```json
{
  "output_file_pattern": "[source_basename]_custom.[default_extension]"
}
```

**Input B** (Template):
```json
{
  "output_file_pattern": "[source_basename]_[output_track].[default_extension]"
}
```

**Result**:
```json
{
  "output_file_pattern": "[source_basename]_custom.[default_extension]"
}
```

**Explanation**: A's value is always used. If A didn't provide this field, the result would be `null` or the field would be omitted (depending on null handling).

---

### 4. deepMerge

**Description**: Recursively merges objects field-by-field. For each field, A's value wins if present, otherwise B's value is used. Similar to `mergeRight` but explicitly recursive.

**When to Use**:
- Nested configuration objects
- Complex hierarchical data structures
- When you want field-level granularity in nested objects

**Example**:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "database": {
      "type": "object",
      "x-kfs-merge": {"strategy": "deepMerge"},
      "properties": {
        "host": {"type": "string"},
        "port": {"type": "integer"},
        "ssl": {"type": "boolean"}
      }
    }
  }
}
```

**Input A** (API Request):
```json
{
  "database": {
    "port": 5433
  }
}
```

**Input B** (Template):
```json
{
  "database": {
    "host": "localhost",
    "port": 5432,
    "ssl": true
  }
}
```

**Result**:
```json
{
  "database": {
    "host": "localhost",
    "port": 5433,
    "ssl": true
  }
}
```

**Explanation**: The `database` object is merged field-by-field. A's `port` (5433) overrides B's, while `host` and `ssl` from B are preserved since A doesn't specify them.

---

### 5. replace

**Description**: Completely replaces B's value with A's value. No merging occurs. This is the default strategy for arrays.

**When to Use**:
- Arrays where you want complete replacement
- When partial updates don't make sense
- Lists that should be treated atomically

**Example**:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "kfsp_packages": {
      "type": "array",
      "items": {"type": "string"},
      "x-kfs-merge": {"strategy": "replace"}
    }
  }
}
```

**Input A** (API Request):
```json
{
  "kfsp_packages": ["package-v2.0", "addon-v1.5"]
}
```

**Input B** (Template):
```json
{
  "kfsp_packages": ["package-v1.0", "core-v1.0", "utils-v1.0"]
}
```

**Result**:
```json
{
  "kfsp_packages": ["package-v2.0", "addon-v1.5"]
}
```

**Explanation**: A's array completely replaces B's array. None of B's items are preserved. If A doesn't provide the field, B's array is used.

---

### 6. concat

**Description**: Concatenates arrays by appending A's items to B's items. Order: B's items first, then A's items.

**When to Use**:
- Additive lists (tags, labels, flags)
- Accumulating values from both sources
- When both A and B contribute items

**Example**:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "tags": {
      "type": "array",
      "items": {"type": "string"},
      "x-kfs-merge": {"strategy": "concat"}
    }
  }
}
```

**Input A** (API Request):
```json
{
  "tags": ["user-provided", "custom"]
}
```

**Input B** (Template):
```json
{
  "tags": ["default", "system"]
}
```

**Result**:
```json
{
  "tags": ["default", "system", "user-provided", "custom"]
}
```

**Explanation**: B's tags come first, followed by A's tags. Duplicates are allowed. Total length is the sum of both arrays.

---

### 7. concatUnique

**Description**: Concatenates arrays like `concat`, but removes duplicate primitive values. Non-primitive values (objects, arrays) are always included.

**When to Use**:
- Tag arrays where duplicates are meaningless
- Unique identifiers or names
- When you want additive behavior without redundancy

**Example**:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "tags": {
      "type": "array",
      "items": {"type": "string"},
      "x-kfs-merge": {"strategy": "concatUnique"}
    }
  }
}
```

**Input A** (API Request):
```json
{
  "tags": ["production", "urgent", "custom"]
}
```

**Input B** (Template):
```json
{
  "tags": ["production", "default", "system"]
}
```

**Result**:
```json
{
  "tags": ["production", "default", "system", "urgent", "custom"]
}
```

**Explanation**: Arrays are concatenated (B first, then A), but "production" appears only once. First occurrence is kept, duplicates are removed.

---

### 8. mergeByKey

**Description**: Merges arrays of objects by matching them on a specified key field. Objects with matching keys are deep merged; objects unique to A or B are included in the result.

**When to Use**:
- Arrays of configuration objects with unique identifiers
- Dependency lists with name/id fields
- Any array where items have a natural key

**Example**:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "audio_output_config": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "output_track": {"type": "integer"},
          "language": {"type": "string"},
          "output_file_pattern": {"type": "string"}
        }
      },
      "x-kfs-merge": {
        "strategy": "mergeByKey",
        "mergeKey": "output_track"
      }
    }
  }
}
```

**Input A** (API Request):
```json
{
  "audio_output_config": [
    {"output_track": 0, "language": "Spa"},
    {"output_track": 2, "language": "Fra"}
  ]
}
```

**Input B** (Template):
```json
{
  "audio_output_config": [
    {"output_track": 0, "language": "Eng", "output_file_pattern": "[source_basename]_[output_track].[default_extension]"},
    {"output_track": 1, "language": "Eng", "output_file_pattern": "[source_basename]_[output_track].[default_extension]"}
  ]
}
```

**Result**:
```json
{
  "audio_output_config": [
    {
      "output_track": 0,
      "language": "Spa",
      "output_file_pattern": "[source_basename]_[output_track].[default_extension]"
    },
    {
      "output_track": 2,
      "language": "Fra"
    },
    {
      "output_track": 1,
      "language": "Eng",
      "output_file_pattern": "[source_basename]_[output_track].[default_extension]"
    }
  ]
}
```

**Explanation**:
- Track 0: Merged (A's language "Spa" overrides B's "Eng", B's pattern is preserved)
- Track 2: From A only (new track)
- Track 1: From B only (preserved)

#### replaceOnMatch Option

By default, `mergeByKey` deep merges matching items (A's fields override B's, but B's other fields are preserved). Set `replaceOnMatch: true` to completely replace B's item with A's item instead:

```json
{
  "x-kfs-merge": {
    "strategy": "mergeByKey",
    "mergeKey": "output_track",
    "replaceOnMatch": true
  }
}
```

With `replaceOnMatch: true`, when A has `{"output_track": 0, "language": "Spa"}` and B has `{"output_track": 0, "language": "Eng", "output_file_pattern": "..."}`, the result will be `{"output_track": 0, "language": "Spa"}` — B's `output_file_pattern` is NOT preserved.

---

### 9. mergeByDiscriminator

**Description**: Merges arrays of discriminated union objects (oneOf/anyOf) by matching on a discriminator field (typically "type"). Objects with matching discriminator values are deep merged.

**When to Use**:
- Arrays of polymorphic objects with a type field
- Filter chains, plugin configurations
- Any oneOf/anyOf array with a discriminator

**Example**:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "filters": {
      "type": "array",
      "items": {
        "oneOf": [
          {
            "type": "object",
            "properties": {
              "type": {"const": "hqdn3d"},
              "luma_spatial": {"type": "number"},
              "luma_tmp": {"type": "number"}
            }
          },
          {
            "type": "object",
            "properties": {
              "type": {"const": "unsharp"},
              "luma_amount": {"type": "number"},
              "luma_msize_x": {"type": "integer"}
            }
          }
        ]
      },
      "x-kfs-merge": {
        "strategy": "mergeByDiscriminator",
        "discriminatorField": "type"
      }
    }
  }
}
```

**Input A** (API Request):
```json
{
  "filters": [
    {"type": "hqdn3d", "luma_spatial": 10},
    {"type": "sharpen", "amount": 0.5}
  ]
}
```

**Input B** (Template):
```json
{
  "filters": [
    {"type": "hqdn3d", "luma_spatial": 8, "luma_tmp": 6},
    {"type": "unsharp", "luma_amount": -0.98, "luma_msize_x": 3}
  ]
}
```

**Result**:
```json
{
  "filters": [
    {"type": "hqdn3d", "luma_spatial": 10, "luma_tmp": 6},
    {"type": "sharpen", "amount": 0.5},
    {"type": "unsharp", "luma_amount": -0.98, "luma_msize_x": 3}
  ]
}
```

**Explanation**:
- `hqdn3d`: Merged (A's luma_spatial=10 overrides B's 8, B's luma_tmp=6 preserved)
- `sharpen`: From A only (new filter type)
- `unsharp`: From B only (preserved)

#### replaceOnMatch Option

By default, `mergeByDiscriminator` deep merges matching items (A's fields override B's, but B's other fields are preserved). Set `replaceOnMatch: true` to completely replace B's item with A's item instead:

```json
{
  "x-kfs-merge": {
    "strategy": "mergeByDiscriminator",
    "discriminatorField": "type",
    "replaceOnMatch": true
  }
}
```

With `replaceOnMatch: true`, when A has `{"type": "hqdn3d", "luma_spatial": 10}` and B has `{"type": "hqdn3d", "luma_spatial": 8, "luma_tmp": 6}`, the result will be `{"type": "hqdn3d", "luma_spatial": 10}` — B's `luma_tmp` is NOT preserved.

---

### 10. overlay

**Description**: Applies A as a partial update (PATCH-like). Only fields explicitly present in A are applied to B. Fields not in A remain unchanged from B. Respects `nullHandling: "asAbsent"`.

**When to Use**:
- PATCH-style API updates
- Partial configuration overrides
- When you want to distinguish between "not provided" and "set to null"

**Example**:

```json
{
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
}
```

**Input A** (API Request):
```json
{
  "config": {
    "host": "production",
    "debug": true
  }
}
```

**Input B** (Template):
```json
{
  "config": {
    "host": "localhost",
    "port": 5432,
    "timeout": 30,
    "debug": false
  }
}
```

**Result**:
```json
{
  "config": {
    "host": "production",
    "port": 5432,
    "timeout": 30,
    "debug": true
  }
}
```

**Explanation**: A only specifies `host` and `debug`, so only those fields are updated. B's `port` and `timeout` are preserved because A didn't mention them. This differs from `deepMerge` in how it handles null values and absent fields.

---

### 11. sum

**Description**: Adds two numeric values together. If one value is missing, returns the other. Requires both values to be numbers.

**When to Use**:
- Counters and accumulators
- Combining quotas or limits
- Additive numeric values

**Example**:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "total_requests": {
      "type": "integer",
      "x-kfs-merge": {"strategy": "sum"}
    }
  }
}
```

**Input A** (API Request):
```json
{
  "total_requests": 150
}
```

**Input B** (Template):
```json
{
  "total_requests": 50
}
```

**Result**:
```json
{
  "total_requests": 200
}
```

**Explanation**: 150 + 50 = 200. Both values are added together.

---

### 12. max

**Description**: Returns the larger of two numeric values. If one value is missing, returns the other.

**When to Use**:
- Maximum limits or thresholds
- Taking the higher priority value
- Capacity planning

**Example**:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "max_concurrent_requests": {
      "type": "integer",
      "x-kfs-merge": {"strategy": "max"}
    }
  }
}
```

**Input A** (API Request):
```json
{
  "max_concurrent_requests": 8
}
```

**Input B** (Template):
```json
{
  "max_concurrent_requests": 4
}
```

**Result**:
```json
{
  "max_concurrent_requests": 8
}
```

**Explanation**: 8 is greater than 4, so 8 is returned.

---

### 13. min

**Description**: Returns the smaller of two numeric values. If one value is missing, returns the other.

**When to Use**:
- Minimum thresholds or constraints
- Taking the lower priority value
- Resource limits

**Example**:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "min_bitrate_kb": {
      "type": "integer",
      "x-kfs-merge": {"strategy": "min"}
    }
  }
}
```

**Input A** (API Request):
```json
{
  "min_bitrate_kb": 128
}
```

**Input B** (Template):
```json
{
  "min_bitrate_kb": 256
}
```

**Result**:
```json
{
  "min_bitrate_kb": 128
}
```

**Explanation**: 128 is less than 256, so 128 is returned.

---

## Best Practices

### 1. Choose the Right Default Strategy

Set `defaultStrategy` at the root level to establish baseline behavior:

```json
{
  "x-kfs-merge": {
    "defaultStrategy": "mergeRequest",
    "arrayStrategy": "replace"
  }
}
```

- **`mergeRequest`**: Best default for most use cases (request overrides base)
- **`deepMerge`**: Use when you have deeply nested objects
- **`keepBase`**: Use when base/template should be authoritative

### 2. Be Explicit with Arrays

Arrays have different semantics than objects. Always specify array strategy:

```json
{
  "tags": {
    "type": "array",
    "x-kfs-merge": {"strategy": "concatUnique"}  // Explicit!
  }
}
```

Common array strategies:
- **`replace`**: Complete replacement (default)
- **`concat`**: Additive lists
- **`concatUnique`**: Unique additive lists
- **`mergeByKey`**: Arrays of objects with IDs

### 3. Use mergeByKey for Object Arrays

When arrays contain objects with natural keys:

```json
{
  "dependencies": {
    "type": "array",
    "items": {
      "type": "object",
      "properties": {
        "name": {"type": "string"},
        "version": {"type": "string"}
      }
    },
    "x-kfs-merge": {
      "strategy": "mergeByKey",
      "mergeKey": "name"
    }
  }
}
```

### 4. Use mergeByDiscriminator for Polymorphic Arrays

For oneOf/anyOf arrays with a type discriminator:

```json
{
  "filters": {
    "type": "array",
    "items": {
      "oneOf": [...]
    },
    "x-kfs-merge": {
      "strategy": "mergeByDiscriminator",
      "discriminatorField": "type"
    }
  }
}
```

### 5. Understand Null Handling

Control how `null` values are treated:

```json
{
  "x-kfs-merge": {
    "nullHandling": "asAbsent"  // or "asValue"
  }
}
```

- **`asValue`** (default): `null` in A overwrites B's value
- **`asAbsent`**: `null` in A is treated as if the field wasn't provided, B's value is kept

### 6. Layer Strategies for Complex Schemas

Combine strategies at different levels:

```json
{
  "type": "object",
  "x-kfs-merge": {"defaultStrategy": "mergeRequest"},
  "properties": {
    "system_config": {
      "type": "object",
      "x-kfs-merge": {"strategy": "keepBase"}  // Override for this subtree
    },
    "user_config": {
      "type": "object",
      "x-kfs-merge": {"strategy": "overlay"}   // PATCH-like updates
    }
  }
}
```

---

## Common Pitfalls to Avoid

### 1. Forgetting Array Strategy

**Problem**: Arrays default to `replace`, which may not be what you want.

```json
// ❌ BAD: Will replace entire array
{
  "tags": {
    "type": "array",
    "items": {"type": "string"}
  }
}

// ✅ GOOD: Explicit strategy
{
  "tags": {
    "type": "array",
    "items": {"type": "string"},
    "x-kfs-merge": {"strategy": "concatUnique"}
  }
}
```

### 2. Using mergeByKey Without a Key

**Problem**: `mergeByKey` requires `mergeKey` to be specified.

```json
// ❌ BAD: Missing mergeKey
{
  "items": {
    "type": "array",
    "x-kfs-merge": {"strategy": "mergeByKey"}
  }
}

// ✅ GOOD: Specify the key field
{
  "items": {
    "type": "array",
    "x-kfs-merge": {
      "strategy": "mergeByKey",
      "mergeKey": "id"
    }
  }
}
```

### 3. Understanding keepBase and keepRequest

**Semantic Strategy Names**: The strategies use clear semantic names to indicate their behavior.

- **`keepBase`**: Always keeps the base/template (B) value - use for immutable defaults
- **`keepRequest`**: Always keeps the request/override (A) value - use for required user input

```json
// Base/template should win (immutable defaults)
{
  "retry_mode": {
    "type": "string",
    "x-kfs-merge": {"strategy": "keepBase"}  // B wins
  }
}

// Request should always win (user control)
{
  "custom_field": {
    "type": "string",
    "x-kfs-merge": {"strategy": "keepRequest"}  // A wins
  }
}
```

### 4. Mixing Strategies Incorrectly

**Problem**: Using numeric strategies on non-numeric fields.

```json
// ❌ BAD: sum on strings will fail
{
  "name": {
    "type": "string",
    "x-kfs-merge": {"strategy": "sum"}
  }
}

// ✅ GOOD: Use appropriate strategy for type
{
  "name": {
    "type": "string",
    "x-kfs-merge": {"strategy": "mergeRight"}
  },
  "count": {
    "type": "integer",
    "x-kfs-merge": {"strategy": "sum"}
  }
}
```

### 5. Not Understanding Null Handling

**Problem**: Unexpected behavior with `null` values.

```json
// With default nullHandling: "asValue"
// A: {"name": null}
// B: {"name": "template"}
// Result: {"name": null}  ← A's null overwrites B

// With nullHandling: "asAbsent"
// A: {"name": null}
// B: {"name": "template"}
// Result: {"name": "template"}  ← A's null is ignored
```

Choose based on your API semantics:
- **`asValue`**: Use when `null` means "clear this field"
- **`asAbsent`**: Use when `null` means "I'm not specifying this field"

### 6. Overusing overlay

**Problem**: `overlay` is powerful but can be confusing. Use it only when you need PATCH semantics.

```json
// Use overlay when you want partial updates
{
  "config": {
    "x-kfs-merge": {"strategy": "overlay"}
  }
}

// Use deepMerge for standard nested object merging
{
  "config": {
    "x-kfs-merge": {"strategy": "deepMerge"}
  }
}
```

---

## Complete Real-World Example

Here's a comprehensive example using multiple strategies from the media encoding domain:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "x-kfs-merge": {
    "defaultStrategy": "mergeRight",
    "arrayStrategy": "replace",
    "nullHandling": "asAbsent"
  },
  "properties": {
    "job_timeout_seconds": {
      "type": "integer",
      "x-kfs-merge": {"strategy": "max"}
    },
    "backend_settings": {
      "type": "object",
      "properties": {
        "selected": {
          "type": "string"
        },
        "options": {
          "type": "object",
          "x-kfs-merge": {"strategy": "overlay"},
          "properties": {
            "json_runner": {
              "type": "object",
              "properties": {
                "json_path": {"type": "string"},
                "report_level": {
                  "type": "string",
                  "x-kfs-merge": {"strategy": "keepLeft"}
                }
              }
            }
          }
        }
      }
    },
    "audio_output_config": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "output_track": {"type": "integer"},
          "language": {"type": "string"},
          "output_file_pattern": {"type": "string"}
        }
      },
      "x-kfs-merge": {
        "strategy": "mergeByKey",
        "mergeKey": "output_track"
      }
    },
    "tags": {
      "type": "array",
      "items": {"type": "string"},
      "x-kfs-merge": {"strategy": "concatUnique"}
    },
    "deinterlacing_options": {
      "type": "object",
      "properties": {
        "algo": {
          "type": "object",
          "properties": {
            "yadif": {
              "type": "object",
              "properties": {
                "mode": {"type": "integer"},
                "parity": {"type": "number"}
              }
            },
            "bwdif": {
              "type": "object",
              "properties": {
                "mode": {"type": "integer"},
                "deint": {"type": "integer"}
              }
            }
          }
        }
      }
    }
  }
}
```

**Input A** (API Request):
```json
{
  "job_timeout_seconds": 7200,
  "backend_settings": {
    "selected": "json_runner",
    "options": {
      "json_runner": {
        "json_path": "custom_tasks.json",
        "report_level": "high"
      }
    }
  },
  "audio_output_config": [
    {"output_track": 0, "language": "Spa"},
    {"output_track": 2, "language": "Fra"}
  ],
  "tags": ["urgent", "production"],
  "deinterlacing_options": {
    "algo": {
      "yadif": {
        "mode": 1
      }
    }
  }
}
```

**Input B** (Template):
```json
{
  "job_timeout_seconds": 3600,
  "backend_settings": {
    "selected": "local",
    "options": {
      "json_runner": {
        "json_path": "tasks.json",
        "report_level": "medium_high",
        "worker_workdir_basedir_pattern": "${CAE_WORKER_WORK_DIR:-/workdir}"
      },
      "local": {
        "max_workers": 4
      }
    }
  },
  "audio_output_config": [
    {
      "output_track": 0,
      "language": "Eng",
      "output_file_pattern": "[source_basename]_[output_track].[default_extension]"
    },
    {
      "output_track": 1,
      "language": "Eng",
      "output_file_pattern": "[source_basename]_[output_track].[default_extension]"
    }
  ],
  "tags": ["default", "production"],
  "deinterlacing_options": {
    "algo": {
      "yadif": {
        "mode": 0,
        "parity": -1
      }
    },
    "engaging_mode": "auto"
  }
}
```

**Result**:
```json
{
  "job_timeout_seconds": 7200,
  "backend_settings": {
    "selected": "json_runner",
    "options": {
      "json_runner": {
        "json_path": "custom_tasks.json",
        "report_level": "medium_high",
        "worker_workdir_basedir_pattern": "${CAE_WORKER_WORK_DIR:-/workdir}"
      },
      "local": {
        "max_workers": 4
      }
    }
  },
  "audio_output_config": [
    {
      "output_track": 0,
      "language": "Spa",
      "output_file_pattern": "[source_basename]_[output_track].[default_extension]"
    },
    {
      "output_track": 2,
      "language": "Fra"
    },
    {
      "output_track": 1,
      "language": "Eng",
      "output_file_pattern": "[source_basename]_[output_track].[default_extension]"
    }
  ],
  "tags": ["default", "production", "urgent"],
  "deinterlacing_options": {
    "algo": {
      "yadif": {
        "mode": 1,
        "parity": -1
      }
    },
    "engaging_mode": "auto"
  }
}
```

**Explanation**:

1. **`job_timeout_seconds`** (max strategy): 7200 > 3600, so 7200 wins
2. **`backend_settings.selected`** (mergeRight): A's "json_runner" wins
3. **`backend_settings.options`** (overlay strategy):
   - A only specifies `json_runner`, so only that is updated
   - B's `local` is preserved
   - Within `json_runner`: A's `json_path` wins, B's `report_level` wins (keepLeft), B's `worker_workdir_basedir_pattern` preserved
4. **`audio_output_config`** (mergeByKey on output_track):
   - Track 0: Merged (A's language, B's pattern)
   - Track 2: From A (new)
   - Track 1: From B (preserved)
5. **`tags`** (concatUnique): ["default", "production", "urgent"] - "production" appears only once
6. **`deinterlacing_options`** (deepMerge):
   - `algo.yadif.mode`: A's 1 wins
   - `algo.yadif.parity`: B's -1 preserved
   - `engaging_mode`: B's "auto" preserved

---

## Summary

The kfs-flow-merge library provides 13 powerful merge strategies to handle any JSON merging scenario:

| Strategy | Use Case | Key Behavior |
|----------|----------|--------------|
| `mergeRequest` | Default, request overrides | Request (A) wins if present |
| `keepBase` | Immutable base/template | Base (B) always wins |
| `keepRequest` | Required user input | Request (A) always wins |
| `deepMerge` | Nested objects | Recursive field merge |
| `replace` | Arrays (default) | Complete replacement |
| `concat` | Additive arrays | B + A |
| `concatUnique` | Unique tags | B + A, deduplicated |
| `mergeByKey` | Object arrays | Match by key field |
| `mergeByDiscriminator` | Polymorphic arrays | Match by type field |
| `overlay` | PATCH updates | Partial application |
| `sum` | Counters | A + B |
| `max` | Limits | max(A, B) |
| `min` | Thresholds | min(A, B) |

Choose strategies based on your data semantics, and layer them appropriately for complex schemas. Always be explicit with array strategies and understand null handling for predictable results.


