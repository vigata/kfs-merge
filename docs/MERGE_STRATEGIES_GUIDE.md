# KFS-Flow-Merge: Comprehensive Merge Strategies Guide

## Table of Contents

- [Introduction](#introduction)
- [Strategy Reference](#strategy-reference)
  - [1. deepMerge (Default)](#1-deepmerge-default)
  - [2. keepBase](#2-keepbase)
  - [3. keepRequest](#3-keeprequest)
  - [4. replace](#4-replace)
  - [5. concat](#5-concat)
  - [6. mergeByDiscriminator](#6-mergebydiscriminator)
  - [7. numeric](#7-numeric)
- [Best Practices](#best-practices)
- [Common Pitfalls to Avoid](#common-pitfalls-to-avoid)
- [Complete Real-World Example](#complete-real-world-example)
- [Summary](#summary)

---

## Introduction

The kfs-flow-merge library provides 7 distinct merge strategies to control how JSON instances are combined. Each strategy is configured using the `x-kfs-merge` extension in your JSON Schema, allowing fine-grained control over merge behavior at every level of your data structure.

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
    "discriminatorField": "type",     // For mergeByDiscriminator strategy
    "replaceOnMatch": true,           // Default for mergeByDiscriminator (set false to deep merge matches)
    "unique": true,                   // For concat strategy: deduplicate items
    "operation": "sum",               // For numeric strategy: "sum", "max", or "min"
    "defaultStrategy": "deepMerge",   // Default for all fields
    "arrayStrategy": "replace",       // Default for arrays
    "nullHandling": "asAbsent"        // How to treat null values
  }
}
```

---

## Strategy Reference

### 1. deepMerge (Default)

**Description**: Recursively merges objects field-by-field. For each field, A's value wins if present, otherwise B's value is used. Respects `nullHandling` configuration for null values. This is the default strategy for all non-array fields.

**When to Use**:
- Default behavior for most fields
- API requests overriding template defaults
- Nested configuration objects
- Complex hierarchical data structures

**Example**:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "x-kfs-merge": {"defaultStrategy": "deepMerge"},
  "properties": {
    "job_timeout_seconds": {"type": "integer"},
    "max_attempts": {"type": "integer"},
    "database": {
      "type": "object",
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
  "job_timeout_seconds": 3600,
  "database": {
    "port": 5433
  }
}
```

**Input B** (Base/Template):
```json
{
  "job_timeout_seconds": 86400,
  "max_attempts": 10,
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
  "job_timeout_seconds": 3600,
  "max_attempts": 10,
  "database": {
    "host": "localhost",
    "port": 5433,
    "ssl": true
  }
}
```

**Explanation**:
- A's `job_timeout_seconds` (3600) overrides B's value
- A doesn't specify `max_attempts`, so B's value (10) is preserved
- The `database` object is merged field-by-field: A's `port` (5433) overrides B's, while `host` and `ssl` from B are preserved

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

**Explanation**: `retry_mode` uses `keepBase`, so B's "standard" is preserved despite A specifying "aggressive". `max_attempts` uses default `deepMerge`, so A's 20 wins.

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

### 4. replace

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

### 5. concat

**Description**: Concatenates arrays by appending A's items to B's items. Order: B's items first, then A's items. Use the `unique` option to remove duplicate primitive values.

**Options**:
- `unique` (boolean, default: false): When true, removes duplicate primitive values after concatenation. Non-primitive values (objects, arrays) are always included.

**When to Use**:
- Additive lists (tags, labels, flags)
- Accumulating values from both sources
- When both A and B contribute items
- With `unique: true` for tag arrays where duplicates are meaningless

**Example (basic concat)**:

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

**Example (concat with unique)**:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "tags": {
      "type": "array",
      "items": {"type": "string"},
      "x-kfs-merge": {"strategy": "concat", "unique": true}
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

### 6. mergeByDiscriminator

**Description**: Merges arrays of discriminated union objects (oneOf/anyOf) by matching on a discriminator field (typically "type"). By default, matching items are **replaced** (A’s item fully replaces B’s). Set `replaceOnMatch: false` to deep merge matching items instead.

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
    {"type": "hqdn3d", "luma_spatial": 10},
    {"type": "sharpen", "amount": 0.5},
    {"type": "unsharp", "luma_amount": -0.98, "luma_msize_x": 3}
  ]
}
```

**Explanation (replaceOnMatch default = true)**:
- `hqdn3d`: A's item replaces B's, so B's `luma_tmp` is dropped.
- `sharpen`: From A only (new filter type).
- `unsharp`: From B only (preserved).

To deep merge matching items instead of replacing, set `replaceOnMatch: false` (no example shown).

---

### 7. numeric

**Description**: Performs numeric operations on two values. Supports sum, max, and min operations via the `operation` option. If one value is missing, returns the other. Requires both values to be numbers.

**Options**:
- `operation` (string, default: "sum"): The numeric operation to perform. Valid values: "sum", "max", "min"

**When to Use**:
- `sum`: Counters and accumulators, combining quotas or limits
- `max`: Maximum limits or thresholds, taking the higher priority value
- `min`: Minimum thresholds or constraints, resource limits

**Example (sum operation)**:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "total_requests": {
      "type": "integer",
      "x-kfs-merge": {"strategy": "numeric", "operation": "sum"}
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

**Example (max operation)**:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "max_concurrent_requests": {
      "type": "integer",
      "x-kfs-merge": {"strategy": "numeric", "operation": "max"}
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

**Example (min operation)**:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "min_bitrate_kb": {
      "type": "integer",
      "x-kfs-merge": {"strategy": "numeric", "operation": "min"}
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
    "defaultStrategy": "deepMerge",
    "arrayStrategy": "replace"
  }
}
```

- **`deepMerge`**: Best default for most use cases (request overrides base, recursive for objects)
- **`keepBase`**: Use when base/template should be authoritative

### 2. Be Explicit with Arrays

Arrays have different semantics than objects. Always specify array strategy:

```json
{
  "tags": {
    "type": "array",
    "x-kfs-merge": {"strategy": "concat", "unique": true}  // Explicit!
  }
}
```

Common array strategies:
- **`replace`**: Complete replacement (default)
- **`concat`**: Additive lists (use `unique: true` for deduplication)
- **`mergeByDiscriminator`**: Arrays of objects matched by a field

### 3. Use mergeByDiscriminator for Object Arrays

When arrays contain objects with a natural key or discriminator field:

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
      "strategy": "mergeByDiscriminator",
      "discriminatorField": "name"
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
  "x-kfs-merge": {"defaultStrategy": "deepMerge"},
  "properties": {
    "system_config": {
      "type": "object",
      "x-kfs-merge": {"strategy": "keepBase"}  // Override for this subtree
    },
    "user_config": {
      "type": "object",
      "x-kfs-merge": {"strategy": "deepMerge"}   // Recursive merge
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
    "x-kfs-merge": {"strategy": "concat", "unique": true}
  }
}
```

### 2. Understanding keepBase and keepRequest

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
// ❌ BAD: numeric on strings will fail
{
  "name": {
    "type": "string",
    "x-kfs-merge": {"strategy": "numeric", "operation": "sum"}
  }
}

// ✅ GOOD: Use appropriate strategy for type
{
  "name": {
    "type": "string",
    "x-kfs-merge": {"strategy": "deepMerge"}
  },
  "count": {
    "type": "integer",
    "x-kfs-merge": {"strategy": "numeric", "operation": "sum"}
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

---

## Complete Real-World Example

Here's a comprehensive example using multiple strategies from the media encoding domain:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "x-kfs-merge": {
    "defaultStrategy": "deepMerge",
    "arrayStrategy": "replace",
    "nullHandling": "asAbsent"
  },
  "properties": {
    "job_timeout_seconds": {
      "type": "integer",
      "x-kfs-merge": {"strategy": "numeric", "operation": "max"}
    },
    "backend_settings": {
      "type": "object",
      "properties": {
        "selected": {
          "type": "string"
        },
        "options": {
          "type": "object",
          "x-kfs-merge": {"strategy": "deepMerge"},
          "properties": {
            "json_runner": {
              "type": "object",
              "properties": {
                "json_path": {"type": "string"},
                "report_level": {
                  "type": "string",
                  "x-kfs-merge": {"strategy": "keepBase"}
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
        "strategy": "mergeByDiscriminator",
        "discriminatorField": "output_track"
      }
    },
    "tags": {
      "type": "array",
      "items": {"type": "string"},
      "x-kfs-merge": {"strategy": "concat", "unique": true}
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

1. **`job_timeout_seconds`** (numeric with max): 7200 > 3600, so 7200 wins
2. **`backend_settings.selected`** (deepMerge): A's "json_runner" wins
3. **`backend_settings.options`** (deepMerge strategy):
   - A only specifies `json_runner`, so only that is updated
   - B's `local` is preserved
   - Within `json_runner`: A's `json_path` wins, B's `report_level` wins (keepBase), B's `worker_workdir_basedir_pattern` preserved
4. **`audio_output_config`** (mergeByDiscriminator on output_track):
   - Track 0: Merged (A's language, B's pattern)
   - Track 2: From A (new)
   - Track 1: From B (preserved)
5. **`tags`** (concat with unique): ["default", "production", "urgent"] - "production" appears only once
6. **`deinterlacing_options`** (deepMerge):
   - `algo.yadif.mode`: A's 1 wins
   - `algo.yadif.parity`: B's -1 preserved
   - `engaging_mode`: B's "auto" preserved

---

## Summary

The kfs-flow-merge library provides 7 powerful merge strategies to handle any JSON merging scenario:

| Strategy | Use Case | Key Behavior | Options |
|----------|----------|--------------|---------|
| `deepMerge` | Default, request overrides | Recursive merge, A wins on conflict | - |
| `keepBase` | Immutable base/template | Base (B) always wins | - |
| `keepRequest` | Required user input | Request (A) always wins | - |
| `replace` | Arrays (default) | Complete replacement | - |
| `concat` | Additive arrays | B + A | `unique: true` for deduplication |
| `mergeByDiscriminator` | Object arrays | Match by discriminator field | `discriminatorField`, `replaceOnMatch` |
| `numeric` | Counters, limits, thresholds | sum, max, or min of values | `operation: "sum"\|"max"\|"min"` |

Choose strategies based on your data semantics, and layer them appropriately for complex schemas. Always be explicit with array strategies and understand null handling for predictable results.
