// Package kfsmerge provides JSON Schema-based merging of JSON instances.
//
// It validates two JSON instances (A and B) against a schema, merges them
// according to rules embedded in the schema using x-kfs-merge extensions,
// and validates the result.
//
// When merging using Merge(a, b):
//   - A (first parameter) is the request/override instance (typically API request or user input)
//   - B (second parameter) is the base/template instance (typically defaults or template configuration)
//   - By default, A takes precedence over B (request overrides base)
package kfsmerge

// MergeStrategy defines how two values should be merged.
type MergeStrategy string

const (
	// StrategyMergeRequest uses the request's (A) value if present, otherwise the base's (B) value (default).
	StrategyMergeRequest MergeStrategy = "mergeRequest"
	// StrategyKeepBase always uses the base's (B) value, ignoring the request (A).
	StrategyKeepBase MergeStrategy = "keepBase"
	// StrategyKeepRequest always uses the request's (A) value, ignoring the base (B).
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
	DefaultStrategy MergeStrategy `json:"defaultStrategy,omitempty"`
	ArrayStrategy   MergeStrategy `json:"arrayStrategy,omitempty"`
	NullHandling    NullHandling  `json:"nullHandling,omitempty"`
}

// FieldMergeConfig holds per-field merge configuration.
type FieldMergeConfig struct {
	Strategy           MergeStrategy `json:"strategy,omitempty"`
	MergeKey           string        `json:"mergeKey,omitempty"`
	DiscriminatorField string        `json:"discriminatorField,omitempty"`
	ReplaceOnMatch     *bool         `json:"replaceOnMatch,omitempty"`
	NullHandling       NullHandling  `json:"nullHandling,omitempty"`
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

// MergeOptions controls the merge and validation behavior.
type MergeOptions struct {
	SkipValidateA      bool
	SkipValidateB      bool
	SkipValidateResult bool
}

// DefaultMergeOptions returns the default options (all validations enabled).
func DefaultMergeOptions() MergeOptions {
	return MergeOptions{}
}

// ValidationPhase indicates which stage of the merge process a validation error occurred.
type ValidationPhase string

const (
	PhaseValidateA      ValidationPhase = "validate_a"
	PhaseValidateB      ValidationPhase = "validate_b"
	PhaseValidateResult ValidationPhase = "validate_result"
)

