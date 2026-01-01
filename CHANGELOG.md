# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.0.0] - 2025-12-29

### Breaking Changes

**Strategy Naming Convention Update**: All merge strategy names have been renamed to use semantic terminology instead of positional terminology. This is a **breaking change** that requires updating all schema files.

#### Strategy Renames

| Old Name (v1.x) | New Name (v2.0) | Description |
|-----------------|-----------------|-------------|
| `mergeRight` | `mergeRequest` | Request (A) wins if present, else base (B) |
| `keepLeft` | `keepBase` | Always use base (B) value |
| `keepRight` | `keepRequest` | Always use request (A) value |

All other strategies (`deepMerge`, `replace`, `concat`, `concatUnique`, `mergeByDiscriminator`, `overlay`, `sum`, `max`, `min`) remain unchanged.

#### Migration Guide

To migrate from v1.x to v2.0:

1. **Update all JSON schema files** that use the old strategy names:
   ```bash
   # Use this script to update all JSON files in your project
   find . -name "*.json" -type f -exec sed -i '' \
     -e 's/"mergeRight"/"mergeRequest"/g' \
     -e 's/"keepLeft"/"keepBase"/g' \
     -e 's/"keepRight"/"keepRequest"/g' \
     {} +
   ```

2. **Review the changes** to ensure they're correct:
   ```bash
   git diff
   ```

3. **Update your Go module**:
   ```bash
   go get github.com/nbcuni/kfs-flow-merge@v2.0.0
   ```

4. **Run your tests** to verify everything works correctly.

#### Rationale

The old naming convention (`keepLeft`/`keepRight`/`mergeRight`) was based on parameter position in the `Merge(a, b)` function, which was counterintuitive:
- "Right" referred to the first parameter (a)
- "Left" referred to the second parameter (b)

The new naming convention uses semantic names that clearly describe what the data represents:
- `keepBase` - keeps the base/template value (parameter b)
- `keepRequest` - keeps the request/override value (parameter a)
- `mergeRequest` - request wins if present, else base

This makes the code more readable and eliminates confusion about which parameter is which.

### Changed

- **Documentation**: Updated all documentation to use the new semantic terminology (request/base instead of left/right)
- **Examples**: Updated all example schema files to use new strategy names
- **Package Documentation**: Updated package-level and function-level documentation to clarify parameter roles
- **CLI Help Text**: Updated CLI usage text to use semantic terminology

### Internal Changes

- Renamed `StrategyMergeRight` constant to `StrategyMergeRequest`
- Renamed `StrategyKeepLeft` constant to `StrategyKeepBase`
- Renamed `StrategyKeepRight` constant to `StrategyKeepRequest`
- Renamed `mergeRight()` method to `mergeRequest()` in merger implementation
- Updated all test cases to use new strategy names
- Updated all code comments to use semantic terminology

## [1.0.0] - 2025-12-XX

### Added

- Initial release of kfs-flow-merge library
- JSON Schema Draft 2020-12 validation support
- 12 merge strategies: mergeRight, keepLeft, keepRight, deepMerge, replace, concat, concatUnique, mergeByDiscriminator, overlay, sum, max, min
- Configurable null handling (asValue, asAbsent, preserve)
- CLI tool for merging JSON instances
- Comprehensive documentation and examples
- Support for $ref and $defs in schemas
- Per-field merge strategy configuration via x-kfs-merge extension

