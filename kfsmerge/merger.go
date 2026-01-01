package kfsmerge

// Merger merges two JSON instances according to schema-defined rules.
type Merger struct {
	schema *Schema
}

// NewMerger creates a new Merger for the given schema.
func NewMerger(s *Schema) *Merger {
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
	a, b = m.handleNulls(a, b, path)
	config := m.getFieldConfig(a, path)

	switch config.Strategy {
	case StrategyKeepBase:
		return b, nil
	case StrategyKeepRequest:
		return a, nil
	case StrategyDeepMerge:
		return m.deepMerge(a, b, path)
	case StrategyReplace:
		if a != nil {
			return a, nil
		}
		return b, nil
	case StrategyConcat:
		return m.concatArrays(a, b, config.UniqueOrDefault())
	case StrategyMergeByDiscriminator:
		return m.mergeByDiscriminator(a, b, config.DiscriminatorField, config.ReplaceOnMatchOrDefault(), path)
	case StrategyNumeric:
		return m.numericOperation(a, b, config.OperationOrDefault())
	default:
		return m.deepMerge(a, b, path)
	}
}

// getFieldConfig determines the merge configuration for a given path.
func (m *Merger) getFieldConfig(a any, path string) FieldMergeConfig {
	if config, ok := m.schema.FieldConfig(path); ok && config.Strategy != "" {
		return config
	}

	globalConfig := m.schema.GlobalConfig()
	if _, isArray := a.([]any); isArray {
		return FieldMergeConfig{Strategy: globalConfig.ArrayStrategy}
	}

	return FieldMergeConfig{Strategy: globalConfig.DefaultStrategy}
}

// deepMerge recursively merges two values. For objects, it merges field-by-field.
// For scalars, A wins if present. Respects nullHandling configuration.
func (m *Merger) deepMerge(a, b any, path string) (any, error) {
	aMap, aIsMap := a.(map[string]any)
	bMap, bIsMap := b.(map[string]any)

	// If both are objects, merge field-by-field
	if aIsMap && bIsMap {
		result := make(map[string]any)
		for k, v := range bMap {
			result[k] = v
		}

		for k, aVal := range aMap {
			fieldPath := path + "/" + k
			bVal, bHasKey := bMap[k]

			if !bHasKey {
				result[k] = aVal
			} else {
				merged, err := m.mergeValues(aVal, bVal, fieldPath)
				if err != nil {
					return nil, err
				}
				result[k] = merged
			}
		}

		return result, nil
	}

	// For non-objects (scalars, arrays, mixed types): A wins if present
	// Respect nullHandling for null values
	if a == nil {
		nullHandling := m.schema.NullHandlingFor(path)
		if nullHandling == NullAsAbsent {
			// Treat null as absent - B wins
			return b, nil
		}
		// NullAsValue or NullPreserve: null is a value, A (null) wins
		return nil, nil
	}

	return a, nil
}

// handleNulls adjusts A and B values based on null handling configuration.
func (m *Merger) handleNulls(a, b any, path string) (any, any) {
	nullHandling := m.schema.NullHandlingFor(path)

	switch nullHandling {
	case NullAsAbsent:
		if a == nil {
			return nil, b
		}
		if b == nil {
			return a, nil
		}
	case NullPreserve:
		return a, b
	case NullAsValue:
		return a, b
	}

	return a, b
}
