// Command kfsmerge is a CLI tool for merging JSON instances according to a schema.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/nbcuni/kfs-flow-merge/kfsmerge"
	"github.com/spf13/cobra"
)

var (
	schemaPath       string
	instanceAPath    string
	instanceBPath    string
	outputPath       string
	skipValidateA    bool
	skipValidateB    bool
	skipValidateR    bool
	applyDefaultsStr string
	pretty           bool
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "kfsmerge",
	Short: "Merge JSON instances according to a schema",
	Long: `Merge two JSON instances according to a schema with x-kfs-merge rules.
Instance A (request/override) is merged with B (base/template), with A taking precedence.`,
	RunE: runMerge,
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate JSON instances against the schema",
	Long:  `Validate one or more JSON instances against the schema without merging.`,
	RunE:  runValidate,
}

func init() {
	// Root command flags
	rootCmd.PersistentFlags().StringVarP(&schemaPath, "schema", "s", "", "Path to JSON Schema file (required)")
	rootCmd.PersistentFlags().StringVarP(&instanceAPath, "instance-a", "a", "", "Path to instance A JSON file")
	rootCmd.PersistentFlags().StringVarP(&instanceBPath, "instance-b", "b", "", "Path to instance B JSON file")
	rootCmd.MarkPersistentFlagRequired("schema")

	// Merge-specific flags
	rootCmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output file path (default: stdout)")
	rootCmd.Flags().BoolVar(&skipValidateA, "skip-validate-a", false, "Skip validation of instance A")
	rootCmd.Flags().BoolVar(&skipValidateB, "skip-validate-b", false, "Skip validation of instance B")
	rootCmd.Flags().BoolVar(&skipValidateR, "skip-validate-result", false, "Skip validation of result")
	rootCmd.Flags().StringVar(&applyDefaultsStr, "apply-defaults", "", "Apply schema default values: true, false, or empty to use schema setting")
	rootCmd.Flags().BoolVar(&pretty, "pretty", true, "Pretty-print JSON output")

	// Add subcommands
	rootCmd.AddCommand(validateCmd)
}

func runMerge(cmd *cobra.Command, args []string) error {
	if instanceAPath == "" || instanceBPath == "" {
		return fmt.Errorf("both --instance-a (-a) and --instance-b (-b) are required for merge")
	}

	// Load schema
	schema, err := kfsmerge.LoadSchemaFromFile(schemaPath)
	if err != nil {
		return fmt.Errorf("error loading schema: %w", err)
	}

	// Read instances
	aData, err := os.ReadFile(instanceAPath)
	if err != nil {
		return fmt.Errorf("error reading instance A: %w", err)
	}

	bData, err := os.ReadFile(instanceBPath)
	if err != nil {
		return fmt.Errorf("error reading instance B: %w", err)
	}

	// Build merge options
	opts := kfsmerge.MergeOptions{
		SkipValidateA:      skipValidateA,
		SkipValidateB:      skipValidateB,
		SkipValidateResult: skipValidateR,
	}

	// Set ApplyDefaults if the flag was explicitly provided
	switch applyDefaultsStr {
	case "true":
		t := true
		opts.ApplyDefaults = &t
	case "false":
		f := false
		opts.ApplyDefaults = &f
	case "":
		// Use schema setting (leave as nil)
	default:
		return fmt.Errorf("--apply-defaults must be 'true', 'false', or empty")
	}

	// Merge
	result, err := schema.MergeWithOptions(aData, bData, opts)
	if err != nil {
		return fmt.Errorf("merge failed: %w", err)
	}

	// Format output
	var output []byte
	if pretty {
		var v any
		if err := json.Unmarshal(result, &v); err == nil {
			output, _ = json.MarshalIndent(v, "", "  ")
		} else {
			output = result
		}
	} else {
		output = result
	}

	// Write output
	if outputPath != "" {
		if err := os.WriteFile(outputPath, output, 0644); err != nil {
			return fmt.Errorf("error writing output: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Result written to %s\n", outputPath)
	} else {
		fmt.Println(string(output))
	}

	return nil
}

func runValidate(cmd *cobra.Command, args []string) error {
	// Load schema
	schema, err := kfsmerge.LoadSchemaFromFile(schemaPath)
	if err != nil {
		return fmt.Errorf("error loading schema: %w", err)
	}

	hasError := false

	if instanceAPath != "" {
		if err := validateFile(schema, instanceAPath); err != nil {
			fmt.Fprintf(os.Stderr, "Instance A validation failed: %v\n", err)
			hasError = true
		} else {
			fmt.Println("Instance A: valid")
		}
	}

	if instanceBPath != "" {
		if err := validateFile(schema, instanceBPath); err != nil {
			fmt.Fprintf(os.Stderr, "Instance B validation failed: %v\n", err)
			hasError = true
		} else {
			fmt.Println("Instance B: valid")
		}
	}

	if instanceAPath == "" && instanceBPath == "" {
		return fmt.Errorf("at least one of --instance-a (-a) or --instance-b (-b) is required")
	}

	if hasError {
		os.Exit(1)
	}

	return nil
}

func validateFile(schema *kfsmerge.Schema, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	return schema.Validate(data)
}
