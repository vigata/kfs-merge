package schema

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// MergeExtensionKey is the JSON Schema extension key for merge rules.
const MergeExtensionKey = "x-kfs-merge"

// Schema represents a parsed JSON Schema with merge extensions.
type Schema struct {
	// compiled is the compiled JSON Schema for validation.
	compiled *jsonschema.Schema
	// raw is the raw schema JSON for extension parsing.
	raw map[string]any
	// globalConfig holds schema-level merge configuration.
	globalConfig GlobalMergeConfig
	// fieldConfigs holds per-field merge configurations, keyed by JSON pointer.
	fieldConfigs map[string]FieldMergeConfig
	// defConfigs holds merge configurations from $defs, keyed by definition name.
	defConfigs map[string]FieldMergeConfig
	// refToDefName maps instance paths to their $defs type name for config lookup.
	refToDefName map[string]string
}

// Load parses a JSON Schema with x-kfs-merge extensions.
func Load(schemaJSON []byte) (*Schema, error) {
	// Parse raw JSON to extract extensions
	var raw map[string]any
	if err := json.Unmarshal(schemaJSON, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse schema JSON: %w", err)
	}

	// Use jsonschema.UnmarshalJSON to get the properly typed value
	schemaValue, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema: %w", err)
	}

	// Compile schema for validation
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("schema.json", schemaValue); err != nil {
		return nil, fmt.Errorf("failed to add schema resource: %w", err)
	}

	compiled, err := compiler.Compile("schema.json")
	if err != nil {
		return nil, fmt.Errorf("failed to compile schema: %w", err)
	}

	s := &Schema{
		compiled:     compiled,
		raw:          raw,
		globalConfig: DefaultGlobalConfig(),
		fieldConfigs: make(map[string]FieldMergeConfig),
		defConfigs:   make(map[string]FieldMergeConfig),
		refToDefName: make(map[string]string),
	}

	// Parse global merge config
	if err := s.parseGlobalConfig(); err != nil {
		return nil, fmt.Errorf("failed to parse global merge config: %w", err)
	}

	// Parse $defs first so we can reference them
	if err := s.parseDefsConfigs(); err != nil {
		return nil, fmt.Errorf("failed to parse $defs merge configs: %w", err)
	}

	// Parse per-field merge configs (including $ref resolution)
	if err := s.parseFieldConfigs("", raw); err != nil {
		return nil, fmt.Errorf("failed to parse field merge configs: %w", err)
	}

	return s, nil
}

// parseGlobalConfig extracts the schema-level x-kfs-merge configuration.
func (s *Schema) parseGlobalConfig() error {
	mergeRaw, ok := s.raw[MergeExtensionKey]
	if !ok {
		return nil // No global config, use defaults
	}

	mergeMap, ok := mergeRaw.(map[string]any)
	if !ok {
		return fmt.Errorf("%s must be an object", MergeExtensionKey)
	}

	if strategy, ok := mergeMap["defaultStrategy"].(string); ok {
		s.globalConfig.DefaultStrategy = MergeStrategy(strategy)
	}
	if strategy, ok := mergeMap["arrayStrategy"].(string); ok {
		s.globalConfig.ArrayStrategy = MergeStrategy(strategy)
	}
	if nullHandling, ok := mergeMap["nullHandling"].(string); ok {
		s.globalConfig.NullHandling = NullHandling(nullHandling)
	}

	return nil
}

// parseDefsConfigs extracts merge configurations from $defs.
func (s *Schema) parseDefsConfigs() error {
	defs, ok := s.raw["$defs"].(map[string]any)
	if !ok {
		return nil // No $defs, nothing to do
	}

	for defName, defValue := range defs {
		defMap, ok := defValue.(map[string]any)
		if !ok {
			continue
		}

		// Check for x-kfs-merge at the type level
		if mergeRaw, ok := defMap[MergeExtensionKey]; ok {
			mergeMap, ok := mergeRaw.(map[string]any)
			if !ok {
				return fmt.Errorf("%s in $defs/%s must be an object", MergeExtensionKey, defName)
			}

			config := FieldMergeConfig{}
			if strategy, ok := mergeMap["strategy"].(string); ok {
				config.Strategy = MergeStrategy(strategy)
			}
			if mergeKey, ok := mergeMap["mergeKey"].(string); ok {
				config.MergeKey = mergeKey
			}
			if discriminatorField, ok := mergeMap["discriminatorField"].(string); ok {
				config.DiscriminatorField = discriminatorField
			}
			if replaceOnMatch, ok := mergeMap["replaceOnMatch"].(bool); ok {
				config.ReplaceOnMatch = &replaceOnMatch
			}
			if nullHandling, ok := mergeMap["nullHandling"].(string); ok {
				config.NullHandling = NullHandling(nullHandling)
			}
			s.defConfigs[defName] = config
		}

		// Also parse nested properties within the definition
		if err := s.parseDefFieldConfigs(defName, "", defMap); err != nil {
			return err
		}
	}

	return nil
}

// parseDefFieldConfigs parses field configs within a $defs definition.
// It stores configs keyed by "defName:fieldPath" for later lookup.
func (s *Schema) parseDefFieldConfigs(defName, path string, node map[string]any) error {
	// Check for x-kfs-merge at this level (skip root of def, handled by parseDefsConfigs)
	if path != "" {
		if mergeRaw, ok := node[MergeExtensionKey]; ok {
			mergeMap, ok := mergeRaw.(map[string]any)
			if !ok {
				return fmt.Errorf("%s in $defs/%s%s must be an object", MergeExtensionKey, defName, path)
			}

			config := FieldMergeConfig{}
			if strategy, ok := mergeMap["strategy"].(string); ok {
				config.Strategy = MergeStrategy(strategy)
			}
			if mergeKey, ok := mergeMap["mergeKey"].(string); ok {
				config.MergeKey = mergeKey
			}
			if discriminatorField, ok := mergeMap["discriminatorField"].(string); ok {
				config.DiscriminatorField = discriminatorField
			}
			if replaceOnMatch, ok := mergeMap["replaceOnMatch"].(bool); ok {
				config.ReplaceOnMatch = &replaceOnMatch
			}
			if nullHandling, ok := mergeMap["nullHandling"].(string); ok {
				config.NullHandling = NullHandling(nullHandling)
			}
			// Store with defName:path key for lookup
			s.defConfigs[defName+":"+path] = config
		}
	}

	// Recurse into properties
	if props, ok := node["properties"].(map[string]any); ok {
		for propName, propValue := range props {
			propPath := path + "/" + propName
			if propMap, ok := propValue.(map[string]any); ok {
				if err := s.parseDefFieldConfigs(defName, propPath, propMap); err != nil {
					return err
				}
			}
		}
	}

	// Recurse into items
	if items, ok := node["items"].(map[string]any); ok {
		itemsPath := path + "/items"
		if err := s.parseDefFieldConfigs(defName, itemsPath, items); err != nil {
			return err
		}
	}

	return nil
}

// resolveRef resolves a $ref string to the definition name.
// Returns the definition name and true if it's a local $defs reference.
func (s *Schema) resolveRef(ref string) (string, bool) {
	// Handle local $defs references like "#/$defs/SomeType"
	const defsPrefix = "#/$defs/"
	if len(ref) > len(defsPrefix) && ref[:len(defsPrefix)] == defsPrefix {
		return ref[len(defsPrefix):], true
	}
	return "", false
}

// parseFieldConfigs recursively extracts per-field x-kfs-merge configurations.
func (s *Schema) parseFieldConfigs(path string, node map[string]any) error {
	// Check for $ref and track the mapping
	if ref, ok := node["$ref"].(string); ok {
		if defName, isLocal := s.resolveRef(ref); isLocal {
			s.refToDefName[path] = defName

			// If the $ref target has a merge config, apply it to this path
			if config, ok := s.defConfigs[defName]; ok {
				// Only set if not already set (direct config takes precedence)
				if _, exists := s.fieldConfigs[path]; !exists {
					s.fieldConfigs[path] = config
				}
			}
		}
	}

	// Check for x-kfs-merge at this level
	if mergeRaw, ok := node[MergeExtensionKey]; ok {
		if path != "" { // Skip root level (that's global config)
			mergeMap, ok := mergeRaw.(map[string]any)
			if !ok {
				return fmt.Errorf("%s at %s must be an object", MergeExtensionKey, path)
			}

			config := FieldMergeConfig{}
			if strategy, ok := mergeMap["strategy"].(string); ok {
				config.Strategy = MergeStrategy(strategy)
			}
			if mergeKey, ok := mergeMap["mergeKey"].(string); ok {
				config.MergeKey = mergeKey
			}
			if discriminatorField, ok := mergeMap["discriminatorField"].(string); ok {
				config.DiscriminatorField = discriminatorField
			}
			if replaceOnMatch, ok := mergeMap["replaceOnMatch"].(bool); ok {
				config.ReplaceOnMatch = &replaceOnMatch
			}
			if nullHandling, ok := mergeMap["nullHandling"].(string); ok {
				config.NullHandling = NullHandling(nullHandling)
			}
			s.fieldConfigs[path] = config
		}
	}

	// Handle anyOf - check for $ref in each alternative
	if anyOf, ok := node["anyOf"].([]any); ok {
		for _, alt := range anyOf {
			if altMap, ok := alt.(map[string]any); ok {
				if ref, ok := altMap["$ref"].(string); ok {
					if defName, isLocal := s.resolveRef(ref); isLocal {
						s.refToDefName[path] = defName
						// Apply def config if no direct config exists
						if config, ok := s.defConfigs[defName]; ok {
							if _, exists := s.fieldConfigs[path]; !exists {
								s.fieldConfigs[path] = config
							}
						}
					}
				}
			}
		}
	}

	// Handle oneOf - check for $ref in each alternative
	if oneOf, ok := node["oneOf"].([]any); ok {
		for _, alt := range oneOf {
			if altMap, ok := alt.(map[string]any); ok {
				if ref, ok := altMap["$ref"].(string); ok {
					if defName, isLocal := s.resolveRef(ref); isLocal {
						// For oneOf, track the first def found (discriminated unions
						// will need more sophisticated handling later)
						if _, exists := s.refToDefName[path]; !exists {
							s.refToDefName[path] = defName
						}
					}
				}
			}
		}
	}

	// Recurse into properties
	if props, ok := node["properties"].(map[string]any); ok {
		for propName, propValue := range props {
			propPath := path + "/" + propName
			if propMap, ok := propValue.(map[string]any); ok {
				if err := s.parseFieldConfigs(propPath, propMap); err != nil {
					return err
				}
			}
		}
	}

	// Recurse into items (for arrays)
	if items, ok := node["items"].(map[string]any); ok {
		itemsPath := path + "/items"
		if err := s.parseFieldConfigs(itemsPath, items); err != nil {
			return err
		}
	}

	return nil
}

// GlobalConfig returns the schema-level merge configuration.
func (s *Schema) GlobalConfig() GlobalMergeConfig {
	return s.globalConfig
}

// FieldConfig returns the merge configuration for a specific field path.
// It first checks for direct field configs, then looks up configs from $defs
// based on the path's type reference.
func (s *Schema) FieldConfig(path string) (FieldMergeConfig, bool) {
	// First, check for direct field config
	if config, ok := s.fieldConfigs[path]; ok {
		return config, true
	}

	// Check if this path has a $ref mapping and look up nested def configs
	// Walk up the path to find the closest $ref and compute the relative path
	for basePath := range s.refToDefName {
		if len(path) > len(basePath) && path[:len(basePath)] == basePath {
			defName := s.refToDefName[basePath]
			relativePath := path[len(basePath):]
			// Look up the def config with the relative path
			if config, ok := s.defConfigs[defName+":"+relativePath]; ok {
				return config, true
			}
		}
	}

	return FieldMergeConfig{}, false
}

// NullHandlingFor returns the null handling setting for a specific field path.
// If no field-specific setting exists, returns the global setting.
func (s *Schema) NullHandlingFor(path string) NullHandling {
	if config, ok := s.fieldConfigs[path]; ok && config.NullHandling != "" {
		return config.NullHandling
	}
	return s.globalConfig.NullHandling
}

// CompiledSchema returns the underlying compiled JSON Schema.
func (s *Schema) CompiledSchema() *jsonschema.Schema {
	return s.compiled
}
