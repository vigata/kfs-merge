// Package schema handles JSON Schema loading and parsing with x-kfs-merge extensions.
package schema

// MergeStrategy defines how two values should be merged.
type MergeStrategy string

const (
	// StrategyMergeRequest uses the request's (A) value if present, otherwise the base's (B) value (default).
	// A is typically an API request or user input; B is typically a template or default configuration.
	StrategyMergeRequest MergeStrategy = "mergeRequest"
	// StrategyKeepBase always uses the base's (B) value, ignoring the request (A).
	// Use for immutable template defaults or system-controlled values.
	StrategyKeepBase MergeStrategy = "keepBase"
	// StrategyKeepRequest always uses the request's (A) value, ignoring the base (B).
	// Use when the user must explicitly provide a value with no fallback to defaults.
	StrategyKeepRequest MergeStrategy = "keepRequest"
	// StrategyDeepMerge recursively merges objects.
	StrategyDeepMerge MergeStrategy = "deepMerge"
	// StrategyReplace replaces B's value with A's entirely (default for arrays).
	StrategyReplace MergeStrategy = "replace"
	// StrategyConcat appends A's array items to B's.
	StrategyConcat MergeStrategy = "concat"
	// StrategyConcatUnique appends A's items to B's, removing duplicates.
	StrategyConcatUnique MergeStrategy = "concatUnique"
	// StrategyMergeByKey merges array items by a key field.
	StrategyMergeByKey MergeStrategy = "mergeByKey"
	// StrategyMergeByDiscriminator merges array items by discriminator field (for oneOf unions).
	StrategyMergeByDiscriminator MergeStrategy = "mergeByDiscriminator"
	// StrategyOverlay only applies A's explicitly provided fields to B, preserving B's other fields.
	StrategyOverlay MergeStrategy = "overlay"
	// StrategySum adds numeric values.
	StrategySum MergeStrategy = "sum"
	// StrategyMax takes the larger numeric value.
	StrategyMax MergeStrategy = "max"
	// StrategyMin takes the smaller numeric value.
	StrategyMin MergeStrategy = "min"
)

// NullHandling defines how explicit null values are handled during merge.
type NullHandling string

const (
	// NullAsValue treats explicit null as a value (null overwrites non-null).
	NullAsValue NullHandling = "asValue"
	// NullAsAbsent treats explicit null as if the field is absent.
	NullAsAbsent NullHandling = "asAbsent"
	// NullPreserve preserves null from A if present, otherwise uses B.
	NullPreserve NullHandling = "preserve"
)

// GlobalMergeConfig holds schema-level merge configuration.
type GlobalMergeConfig struct {
	// DefaultStrategy is the default merge strategy for all fields.
	DefaultStrategy MergeStrategy `json:"defaultStrategy,omitempty"`
	// ArrayStrategy is the default strategy for array fields.
	ArrayStrategy MergeStrategy `json:"arrayStrategy,omitempty"`
	// NullHandling controls how explicit null values are handled.
	NullHandling NullHandling `json:"nullHandling,omitempty"`
}

// FieldMergeConfig holds per-field merge configuration.
type FieldMergeConfig struct {
	// Strategy is the merge strategy for this field.
	Strategy MergeStrategy `json:"strategy,omitempty"`
	// MergeKey is the key field name for mergeByKey strategy (arrays of objects).
	MergeKey string `json:"mergeKey,omitempty"`
	// DiscriminatorField is the field name for mergeByDiscriminator strategy (oneOf unions).
	DiscriminatorField string `json:"discriminatorField,omitempty"`
	// ReplaceOnMatch controls behavior when items with matching keys/discriminators are found.
	// When true, A's item completely replaces B's item instead of deep merging them.
	// Applies to mergeByKey and mergeByDiscriminator strategies.
	// Nil means "not specified" so defaults can be applied per-strategy.
	ReplaceOnMatch *bool `json:"replaceOnMatch,omitempty"`
	// NullHandling overrides global null handling for this field.
	NullHandling NullHandling `json:"nullHandling,omitempty"`
}

// DefaultGlobalConfig returns GlobalMergeConfig with default values.
func DefaultGlobalConfig() GlobalMergeConfig {
	return GlobalMergeConfig{
		DefaultStrategy: StrategyMergeRequest,
		ArrayStrategy:   StrategyReplace,
		NullHandling:    NullAsValue,
	}
}

// ReplaceOnMatchOrDefault resolves ReplaceOnMatch with strategy-specific defaults.
// For mergeByKey and mergeByDiscriminator, default is true when unspecified.
// For other strategies, default is false.
func (c FieldMergeConfig) ReplaceOnMatchOrDefault() bool {
	if c.ReplaceOnMatch != nil {
		return *c.ReplaceOnMatch
	}

	switch c.Strategy {
	case StrategyMergeByKey, StrategyMergeByDiscriminator:
		return true
	default:
		return false
	}
}
