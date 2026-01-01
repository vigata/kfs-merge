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
	// StrategyKeepBase always uses the base's (B) value, ignoring the request (A).
	StrategyKeepBase MergeStrategy = "keepBase"
	// StrategyKeepRequest always uses the request's (A) value, ignoring the base (B).
	StrategyKeepRequest MergeStrategy = "keepRequest"
	// StrategyDeepMerge recursively merges objects, with A winning on conflict. Respects nullHandling. (default)
	StrategyDeepMerge MergeStrategy = "deepMerge"
	// StrategyReplace replaces B's value with A's entirely (default for arrays).
	StrategyReplace MergeStrategy = "replace"
	// StrategyConcat appends A's array items to B's. Use Unique option to deduplicate.
	StrategyConcat MergeStrategy = "concat"
	// StrategyMergeByDiscriminator merges array items by discriminator field.
	StrategyMergeByDiscriminator MergeStrategy = "mergeByDiscriminator"
	// StrategyNumeric performs numeric operations (sum, max, min) based on Operation option.
	StrategyNumeric MergeStrategy = "numeric"
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
	ApplyDefaults   bool          `json:"applyDefaults,omitempty"`
}

// FieldMergeConfig holds per-field merge configuration.
type FieldMergeConfig struct {
	Strategy           MergeStrategy `json:"strategy,omitempty"`
	DiscriminatorField string        `json:"discriminatorField,omitempty"`
	ReplaceOnMatch     *bool         `json:"replaceOnMatch,omitempty"`
	NullHandling       NullHandling  `json:"nullHandling,omitempty"`
	Unique             *bool         `json:"unique,omitempty"`    // For concat strategy: deduplicate items
	Operation          string        `json:"operation,omitempty"` // For numeric strategy: "sum", "max", "min"
}

// UniqueOrDefault returns the Unique setting with default false.
func (c FieldMergeConfig) UniqueOrDefault() bool {
	if c.Unique != nil {
		return *c.Unique
	}
	return false
}

// OperationOrDefault returns the Operation setting with default "sum".
func (c FieldMergeConfig) OperationOrDefault() string {
	if c.Operation != "" {
		return c.Operation
	}
	return "sum"
}

// DefaultGlobalConfig returns GlobalMergeConfig with default values.
func DefaultGlobalConfig() GlobalMergeConfig {
	return GlobalMergeConfig{
		DefaultStrategy: StrategyDeepMerge,
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
	case StrategyMergeByDiscriminator:
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
	ApplyDefaults      *bool // nil uses schema setting, non-nil overrides
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
