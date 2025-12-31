// Package merge implements the core merge logic for JSON instances.
package merge

import (
	"github.com/nbcuni/kfs-flow-merge/schema"
)

// Merger merges two JSON instances according to schema-defined rules.
type Merger struct {
	schema *schema.Schema
}

// New creates a new Merger for the given schema.
func New(s *schema.Schema) *Merger {
	return &Merger{schema: s}
}

// Merge merges instance A into instance B according to the schema's merge rules.
// Parameter a is the request/override instance (typically API request or user input).
// Parameter b is the base/template instance (typically defaults or template configuration).
// By default, a takes precedence over b (request overrides base).
func (m *Merger) Merge(a, b any) (any, error) {
	return m.mergeValues(a, b, "")
}

// mergeValues recursively merges two values at the given path.
func (m *Merger) mergeValues(a, b any, path string) (any, error) {
	// Handle null values according to nullHandling config
	a, b = m.handleNulls(a, b, path)

	// Get the merge strategy and config for this path
	config := m.getFieldConfig(a, path)

	switch config.Strategy {
	case schema.StrategyKeepBase:
		return b, nil
	case schema.StrategyKeepRequest:
		return a, nil
	case schema.StrategyMergeRequest:
		return m.mergeRequest(a, b, path)
	case schema.StrategyDeepMerge:
		return m.deepMerge(a, b, path)
	case schema.StrategyReplace:
		if a != nil {
			return a, nil
		}
		return b, nil
	case schema.StrategyConcat:
		return m.concatArrays(a, b)
	case schema.StrategyConcatUnique:
		return m.concatUniqueArrays(a, b)
	case schema.StrategyMergeByKey:
		return m.mergeByKey(a, b, config.MergeKey, config.ReplaceOnMatchOrDefault(), path)
	case schema.StrategyMergeByDiscriminator:
		return m.mergeByDiscriminator(a, b, config.DiscriminatorField, config.ReplaceOnMatchOrDefault(), path)
	case schema.StrategyOverlay:
		return m.overlay(a, b, path)
	case schema.StrategySum:
		return m.sumNumbers(a, b)
	case schema.StrategyMax:
		return m.maxNumber(a, b)
	case schema.StrategyMin:
		return m.minNumber(a, b)
	default:
		return m.mergeRequest(a, b, path)
	}
}

// getFieldConfig determines the merge configuration for a given path.
func (m *Merger) getFieldConfig(a any, path string) schema.FieldMergeConfig {
	// Check for field-specific config
	if config, ok := m.schema.FieldConfig(path); ok && config.Strategy != "" {
		return config
	}

	// Use global defaults based on type
	globalConfig := m.schema.GlobalConfig()
	if _, isArray := a.([]any); isArray {
		return schema.FieldMergeConfig{Strategy: globalConfig.ArrayStrategy}
	}

	return schema.FieldMergeConfig{Strategy: globalConfig.DefaultStrategy}
}

// mergeRequest implements the default merge strategy: request (A) wins if present, else base (B).
// Note: This is called after handleNulls, so a nil value at this point means
// either A didn't have the field, or A had null with asAbsent handling.
func (m *Merger) mergeRequest(a, b any, path string) (any, error) {
	// If both are objects, deep merge them
	aMap, aIsMap := a.(map[string]any)
	bMap, bIsMap := b.(map[string]any)
	if aIsMap && bIsMap {
		return m.deepMerge(aMap, bMap, path)
	}

	// If A is nil, check null handling to decide
	if a == nil {
		nullHandling := m.schema.NullHandlingFor(path)
		if nullHandling == schema.NullAsAbsent {
			// Treat null as absent, use B
			return b, nil
		}
		// For nullAsValue or nullPreserve, return nil (null wins)
		// But if we got here from outside the map iteration, A was truly absent
		// This is handled by the caller passing nil only when field is absent
	}

	// A wins (including explicit null with asValue handling)
	return a, nil
}

// deepMerge recursively merges two objects.
func (m *Merger) deepMerge(a, b any, path string) (any, error) {
	aMap, aIsMap := a.(map[string]any)
	bMap, bIsMap := b.(map[string]any)

	// If not both maps, A wins
	if !aIsMap || !bIsMap {
		if a != nil {
			return a, nil
		}
		return b, nil
	}

	// Start with a copy of B
	result := make(map[string]any)
	for k, v := range bMap {
		result[k] = v
	}

	// Merge A's values into result
	for k, aVal := range aMap {
		fieldPath := path + "/" + k
		bVal, bHasKey := bMap[k]

		if !bHasKey {
			// A has a key B doesn't have
			result[k] = aVal
		} else {
			// Both have the key, merge recursively
			merged, err := m.mergeValues(aVal, bVal, fieldPath)
			if err != nil {
				return nil, err
			}
			result[k] = merged
		}
	}

	return result, nil
}

// handleNulls adjusts A and B values based on null handling configuration.
// Returns the adjusted A and B values.
func (m *Merger) handleNulls(a, b any, path string) (any, any) {
	nullHandling := m.schema.NullHandlingFor(path)

	switch nullHandling {
	case schema.NullAsAbsent:
		// Treat explicit null as if the field is absent
		if a == nil {
			// A is null, act as if A doesn't have this field
			// (use B's value instead)
			return nil, b
		}
		if b == nil {
			// B is null, act as if B doesn't have this field
			return a, nil
		}
	case schema.NullPreserve:
		// If A explicitly has null, preserve it
		// This is the same as asValue for most purposes
		return a, b
	case schema.NullAsValue:
		// Treat null as a value (default behavior)
		// A's null will overwrite B's value
		return a, b
	}

	return a, b
}
