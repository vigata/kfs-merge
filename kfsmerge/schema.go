package kfsmerge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// MergeExtensionKey is the JSON Schema extension key for merge rules.
const MergeExtensionKey = "x-kfs-merge"

// Schema represents a parsed JSON Schema with merge extensions.
type Schema struct {
	compiled     *jsonschema.Schema
	raw          map[string]any
	globalConfig GlobalMergeConfig
	fieldConfigs map[string]FieldMergeConfig
	defConfigs   map[string]FieldMergeConfig
	refToDefName map[string]string
	defaults     map[string]any // cached extracted defaults from schema
}

// LoadSchemaFromFile loads a JSON Schema from a file path.
func LoadSchemaFromFile(path string) (*Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}
	return LoadSchema(data)
}

// LoadSchema parses a JSON Schema with x-kfs-merge extensions.
func LoadSchema(schemaJSON []byte) (*Schema, error) {
	var raw map[string]any
	if err := json.Unmarshal(schemaJSON, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse schema JSON: %w", err)
	}

	schemaValue, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema: %w", err)
	}

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

	if err := s.parseGlobalConfig(); err != nil {
		return nil, fmt.Errorf("failed to parse global merge config: %w", err)
	}

	if err := s.parseDefsConfigs(); err != nil {
		return nil, fmt.Errorf("failed to parse $defs merge configs: %w", err)
	}

	if err := s.parseFieldConfigs("", raw); err != nil {
		return nil, fmt.Errorf("failed to parse field merge configs: %w", err)
	}

	// Pre-extract defaults if applyDefaults is enabled at schema level
	if s.globalConfig.ApplyDefaults {
		s.ExtractDefaults()
	}

	return s, nil
}

// parseGlobalConfig extracts the schema-level x-kfs-merge configuration.
func (s *Schema) parseGlobalConfig() error {
	mergeRaw, ok := s.raw[MergeExtensionKey]
	if !ok {
		return nil
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
	if applyDefaults, ok := mergeMap["applyDefaults"].(bool); ok {
		s.globalConfig.ApplyDefaults = applyDefaults
	}

	return nil
}

// parseDefsConfigs extracts merge configurations from $defs.
func (s *Schema) parseDefsConfigs() error {
	defs, ok := s.raw["$defs"].(map[string]any)
	if !ok {
		return nil
	}

	for defName, defValue := range defs {
		defMap, ok := defValue.(map[string]any)
		if !ok {
			continue
		}

		if mergeRaw, ok := defMap[MergeExtensionKey]; ok {
			mergeMap, ok := mergeRaw.(map[string]any)
			if !ok {
				return fmt.Errorf("%s in $defs/%s must be an object", MergeExtensionKey, defName)
			}

			config := FieldMergeConfig{}
			if strategy, ok := mergeMap["strategy"].(string); ok {
				config.Strategy = MergeStrategy(strategy)
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

		if err := s.parseDefFieldConfigs(defName, "", defMap); err != nil {
			return err
		}
	}

	return nil
}

// parseDefFieldConfigs parses field configs within a $defs definition.
func (s *Schema) parseDefFieldConfigs(defName, path string, node map[string]any) error {
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
			if discriminatorField, ok := mergeMap["discriminatorField"].(string); ok {
				config.DiscriminatorField = discriminatorField
			}
			if replaceOnMatch, ok := mergeMap["replaceOnMatch"].(bool); ok {
				config.ReplaceOnMatch = &replaceOnMatch
			}
			if nullHandling, ok := mergeMap["nullHandling"].(string); ok {
				config.NullHandling = NullHandling(nullHandling)
			}
			s.defConfigs[defName+":"+path] = config
		}
	}

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

	if items, ok := node["items"].(map[string]any); ok {
		itemsPath := path + "/items"
		if err := s.parseDefFieldConfigs(defName, itemsPath, items); err != nil {
			return err
		}
	}

	return nil
}

// resolveRef resolves a $ref string to the definition name.
func (s *Schema) resolveRef(ref string) (string, bool) {
	const defsPrefix = "#/$defs/"
	if len(ref) > len(defsPrefix) && ref[:len(defsPrefix)] == defsPrefix {
		return ref[len(defsPrefix):], true
	}
	return "", false
}

// parseFieldConfigs recursively extracts per-field x-kfs-merge configurations.
func (s *Schema) parseFieldConfigs(path string, node map[string]any) error {
	if ref, ok := node["$ref"].(string); ok {
		if defName, isLocal := s.resolveRef(ref); isLocal {
			s.refToDefName[path] = defName
			if config, ok := s.defConfigs[defName]; ok {
				if _, exists := s.fieldConfigs[path]; !exists {
					s.fieldConfigs[path] = config
				}
			}
		}
	}

	if mergeRaw, ok := node[MergeExtensionKey]; ok {
		if path != "" {
			mergeMap, ok := mergeRaw.(map[string]any)
			if !ok {
				return fmt.Errorf("%s at %s must be an object", MergeExtensionKey, path)
			}

			config := FieldMergeConfig{}
			if strategy, ok := mergeMap["strategy"].(string); ok {
				config.Strategy = MergeStrategy(strategy)
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

	if anyOf, ok := node["anyOf"].([]any); ok {
		for _, alt := range anyOf {
			if altMap, ok := alt.(map[string]any); ok {
				if ref, ok := altMap["$ref"].(string); ok {
					if defName, isLocal := s.resolveRef(ref); isLocal {
						s.refToDefName[path] = defName
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

	if oneOf, ok := node["oneOf"].([]any); ok {
		for _, alt := range oneOf {
			if altMap, ok := alt.(map[string]any); ok {
				if ref, ok := altMap["$ref"].(string); ok {
					if defName, isLocal := s.resolveRef(ref); isLocal {
						if _, exists := s.refToDefName[path]; !exists {
							s.refToDefName[path] = defName
						}
					}
				}
			}
		}
	}

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
func (s *Schema) FieldConfig(path string) (FieldMergeConfig, bool) {
	if config, ok := s.fieldConfigs[path]; ok {
		return config, true
	}

	for basePath := range s.refToDefName {
		if len(path) > len(basePath) && path[:len(basePath)] == basePath {
			defName := s.refToDefName[basePath]
			relativePath := path[len(basePath):]
			if config, ok := s.defConfigs[defName+":"+relativePath]; ok {
				return config, true
			}
		}
	}

	return FieldMergeConfig{}, false
}

// NullHandlingFor returns the null handling setting for a specific field path.
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

// Defaults returns the cached defaults extracted from the schema.
// Returns nil if applyDefaults is not enabled or no defaults exist.
func (s *Schema) Defaults() map[string]any {
	return s.defaults
}

// ExtractDefaults extracts and caches default values from the schema.
// This is called automatically during LoadSchema if applyDefaults is enabled.
func (s *Schema) ExtractDefaults() map[string]any {
	if s.defaults != nil {
		return s.defaults
	}

	defaults := s.extractDefaultsFromNode(s.raw)
	if defaultsMap, ok := defaults.(map[string]any); ok {
		s.defaults = defaultsMap
	}
	return s.defaults
}

// extractDefaultsFromNode recursively extracts defaults from a schema node.
func (s *Schema) extractDefaultsFromNode(node map[string]any) any {
	// Handle $ref first
	if ref, ok := node["$ref"].(string); ok {
		if defName, isLocal := s.resolveRef(ref); isLocal {
			if defs, ok := s.raw["$defs"].(map[string]any); ok {
				if defNode, ok := defs[defName].(map[string]any); ok {
					return s.extractDefaultsFromNode(defNode)
				}
			}
		}
		return nil
	}

	// Get the node's own default value
	nodeDefault := node["default"]

	// Check if this is an object type with properties
	props, hasProps := node["properties"].(map[string]any)
	if !hasProps {
		// Not an object with properties, just return the default
		return nodeDefault
	}

	// Extract defaults from properties (leaf defaults)
	leafDefaults := make(map[string]any)
	for propName, propValue := range props {
		propMap, ok := propValue.(map[string]any)
		if !ok {
			continue
		}

		propDefault := s.extractDefaultsFromNode(propMap)
		if propDefault != nil {
			leafDefaults[propName] = propDefault
		}
	}

	// Merge: leaf defaults override object-level default
	if nodeDefault == nil && len(leafDefaults) == 0 {
		return nil
	}

	if nodeDefault == nil {
		return leafDefaults
	}

	nodeDefaultMap, ok := nodeDefault.(map[string]any)
	if !ok {
		// nodeDefault is not an object, leaf defaults take precedence
		if len(leafDefaults) > 0 {
			return leafDefaults
		}
		return nodeDefault
	}

	// Merge: start with nodeDefaultMap, overlay leafDefaults
	result := make(map[string]any)
	for k, v := range nodeDefaultMap {
		result[k] = v
	}
	for k, v := range leafDefaults {
		result[k] = v
	}

	return result
}
