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
	case StrategyMergeRequest:
		return m.mergeRequest(a, b, path)
	case StrategyDeepMerge:
		return m.deepMerge(a, b, path)
	case StrategyReplace:
		if a != nil {
			return a, nil
		}
		return b, nil
	case StrategyConcat:
		return m.concatArrays(a, b)
	case StrategyConcatUnique:
		return m.concatUniqueArrays(a, b)
	case StrategyMergeByKey:
		return m.mergeByKey(a, b, config.MergeKey, config.ReplaceOnMatchOrDefault(), path)
	case StrategyMergeByDiscriminator:
		return m.mergeByDiscriminator(a, b, config.DiscriminatorField, config.ReplaceOnMatchOrDefault(), path)
	case StrategyOverlay:
		return m.overlay(a, b, path)
	case StrategySum:
		return m.sumNumbers(a, b)
	case StrategyMax:
		return m.maxNumber(a, b)
	case StrategyMin:
		return m.minNumber(a, b)
	default:
		return m.mergeRequest(a, b, path)
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

// mergeRequest implements the default merge strategy: request (A) wins if present, else base (B).
func (m *Merger) mergeRequest(a, b any, path string) (any, error) {
	aMap, aIsMap := a.(map[string]any)
	bMap, bIsMap := b.(map[string]any)
	if aIsMap && bIsMap {
		return m.deepMerge(aMap, bMap, path)
	}

	if a == nil {
		nullHandling := m.schema.NullHandlingFor(path)
		if nullHandling == NullAsAbsent {
			return b, nil
		}
	}

	return a, nil
}

// deepMerge recursively merges two objects.
func (m *Merger) deepMerge(a, b any, path string) (any, error) {
	aMap, aIsMap := a.(map[string]any)
	bMap, bIsMap := b.(map[string]any)

	if !aIsMap || !bIsMap {
		if a != nil {
			return a, nil
		}
		return b, nil
	}

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

