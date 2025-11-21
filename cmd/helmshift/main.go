package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/kdwils/helmshift/pkg/migration"
)

const version = "0.1.0"

func main() {
	// Define flags
	configPath := flag.String("config", "", "Path to migration configuration file (required)")
	valuesPath := flag.String("values", "", "Path to values.yaml file to migrate (required)")
	outputPath := flag.String("output", "", "Path to write migrated values (default: stdout)")
	dryRun := flag.Bool("dry-run", false, "Perform migration but don't write output")
	showVersion := flag.Bool("version", false, "Show version information")
	timeout := flag.Duration("timeout", 30*time.Second, "Timeout for migration execution")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "HelmShift - Helm Values Migration Tool\n\n")
		fmt.Fprintf(os.Stderr, "Usage: helmshift [options]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Migrate values and output to stdout\n")
		fmt.Fprintf(os.Stderr, "  helmshift -config migration.yaml -values old-values.yaml\n\n")
		fmt.Fprintf(os.Stderr, "  # Migrate and write to file\n")
		fmt.Fprintf(os.Stderr, "  helmshift -config migration.yaml -values old-values.yaml -output new-values.yaml\n\n")
		fmt.Fprintf(os.Stderr, "  # Dry run to validate migration\n")
		fmt.Fprintf(os.Stderr, "  helmshift -config migration.yaml -values old-values.yaml -dry-run\n\n")
	}

	flag.Parse()

	if *showVersion {
		fmt.Printf("helmshift version %s\n", version)
		os.Exit(0)
	}

	// Validate required flags
	if *configPath == "" {
		fmt.Fprintf(os.Stderr, "Error: -config is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if *valuesPath == "" {
		fmt.Fprintf(os.Stderr, "Error: -values is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Run migration
	if err := runMigration(*configPath, *valuesPath, *outputPath, *dryRun, *timeout); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runMigration(configPath, valuesPath, outputPath string, dryRun bool, timeout time.Duration) error {
	// Create migration engine
	engine, err := migration.NewEngine(configPath)
	if err != nil {
		return fmt.Errorf("failed to create migration engine: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Perform migration
	fmt.Fprintf(os.Stderr, "Migrating values from %s...\n", valuesPath)
	result, err := engine.Migrate(ctx, valuesPath)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// Display warnings
	if len(result.Warnings) > 0 {
		fmt.Fprintf(os.Stderr, "\nWarnings:\n")
		for _, warning := range result.Warnings {
			fmt.Fprintf(os.Stderr, "  - %s\n", warning)
		}
	}

	// Output result
	if dryRun {
		fmt.Fprintf(os.Stderr, "\nDry run completed successfully (format: %s)\n", result.Format)
		fmt.Fprintf(os.Stderr, "No output written.\n")
		return nil
	}

	if outputPath != "" {
		// Write to file
		if err := os.WriteFile(outputPath, result.Output, 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Migration completed successfully!\n")
		fmt.Fprintf(os.Stderr, "Output written to: %s\n", outputPath)
	} else {
		// Write to stdout
		fmt.Fprintf(os.Stderr, "Migration completed successfully!\n")
		fmt.Fprintf(os.Stderr, "---\n")
		os.Stdout.Write(result.Output)
	}

	return nil
}
