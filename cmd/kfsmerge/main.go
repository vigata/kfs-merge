// Command kfsmerge is a CLI tool for merging JSON instances according to a schema.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	kfsmerge "github.com/nbcuni/kfs-flow-merge"
)

func main() {
	var (
		schemaPath    = flag.String("schema", "", "Path to JSON Schema file (required)")
		instanceAPath = flag.String("a", "", "Path to instance A JSON file (required)")
		instanceBPath = flag.String("b", "", "Path to instance B JSON file (required)")
		outputPath    = flag.String("o", "", "Output file path (default: stdout)")
		skipValidateA = flag.Bool("skip-validate-a", false, "Skip validation of instance A")
		skipValidateB = flag.Bool("skip-validate-b", false, "Skip validation of instance B")
		skipValidateR = flag.Bool("skip-validate-result", false, "Skip validation of result")
		pretty        = flag.Bool("pretty", true, "Pretty-print JSON output")
		validateOnly  = flag.Bool("validate", false, "Validate inputs without merging")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: kfsmerge -schema <schema.json> -a <a.json> -b <b.json> [-o output.json]\n\n")
		fmt.Fprintf(os.Stderr, "Merge two JSON instances according to a schema with x-kfs-merge rules.\n")
		fmt.Fprintf(os.Stderr, "Instance A (request/override) is merged with B (base/template), with A taking precedence.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *schemaPath == "" {
		fmt.Fprintln(os.Stderr, "Error: -schema is required")
		flag.Usage()
		os.Exit(1)
	}

	// Load schema
	schema, err := kfsmerge.LoadSchemaFromFile(*schemaPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading schema: %v\n", err)
		os.Exit(1)
	}

	// Validate-only mode
	if *validateOnly {
		if *instanceAPath != "" {
			if err := validateFile(schema, *instanceAPath, "A"); err != nil {
				fmt.Fprintf(os.Stderr, "Instance A validation failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Instance A: valid")
		}
		if *instanceBPath != "" {
			if err := validateFile(schema, *instanceBPath, "B"); err != nil {
				fmt.Fprintf(os.Stderr, "Instance B validation failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Instance B: valid")
		}
		return
	}

	// Merge mode requires both instances
	if *instanceAPath == "" || *instanceBPath == "" {
		fmt.Fprintln(os.Stderr, "Error: -a and -b are required for merge")
		flag.Usage()
		os.Exit(1)
	}

	// Read instances
	aData, err := os.ReadFile(*instanceAPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading instance A: %v\n", err)
		os.Exit(1)
	}

	bData, err := os.ReadFile(*instanceBPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading instance B: %v\n", err)
		os.Exit(1)
	}

	// Merge
	opts := kfsmerge.MergeOptions{
		SkipValidateA:      *skipValidateA,
		SkipValidateB:      *skipValidateB,
		SkipValidateResult: *skipValidateR,
	}

	result, err := schema.MergeWithOptions(aData, bData, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Merge failed: %v\n", err)
		os.Exit(1)
	}

	// Format output
	var output []byte
	if *pretty {
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
	if *outputPath != "" {
		if err := os.WriteFile(*outputPath, output, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Result written to %s\n", *outputPath)
	} else {
		fmt.Println(string(output))
	}
}

func validateFile(schema *kfsmerge.Schema, path, name string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	return schema.Validate(data)
}
