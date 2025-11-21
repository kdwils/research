package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kdwils/helmshift/pkg/migrator"
)

const version = "0.2.0"

func main() {
	// Define flags
	chartName := flag.String("chart", "", "Chart name (e.g., nginx-ingress) (required)")
	fromVersion := flag.String("from", "", "Current chart version (required)")
	toVersion := flag.String("to", "", "Target chart version (required)")
	valuesPath := flag.String("values", "", "Path to values.yaml file (required)")
	outputPath := flag.String("output", "", "Path to write migrated values (default: stdout)")
	registryURL := flag.String("registry", "https://helmshift-registry.example.com", "Migration registry URL")
	showVersion := flag.Bool("version", false, "Show version information")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "HelmShift v2 - Helm Values Migration Tool\n\n")
		fmt.Fprintf(os.Stderr, "Fetches migrations from remote registry and applies them with zero dependencies.\n\n")
		fmt.Fprintf(os.Stderr, "Usage: helmshift [options]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Migrate nginx-ingress from v3 to v4\n")
		fmt.Fprintf(os.Stderr, "  helmshift -chart nginx-ingress -from 0.1.5 -to 0.3.0 -values values.yaml\n\n")
		fmt.Fprintf(os.Stderr, "  # Use custom registry\n")
		fmt.Fprintf(os.Stderr, "  helmshift -chart mychart -from 1.0.0 -to 2.0.0 -values values.yaml \\\n")
		fmt.Fprintf(os.Stderr, "    -registry https://my-registry.com/migrations\n\n")
		fmt.Fprintf(os.Stderr, "  # Save to file\n")
		fmt.Fprintf(os.Stderr, "  helmshift -chart mychart -from 1.0.0 -to 2.0.0 -values values.yaml \\\n")
		fmt.Fprintf(os.Stderr, "    -output new-values.yaml\n\n")
	}

	flag.Parse()

	if *showVersion {
		fmt.Printf("helmshift version %s\n", version)
		os.Exit(0)
	}

	// Validate required flags
	if *chartName == "" {
		fmt.Fprintf(os.Stderr, "Error: -chart is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if *fromVersion == "" {
		fmt.Fprintf(os.Stderr, "Error: -from is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if *toVersion == "" {
		fmt.Fprintf(os.Stderr, "Error: -to is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if *valuesPath == "" {
		fmt.Fprintf(os.Stderr, "Error: -values is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Run migration
	if err := runMigration(*chartName, *fromVersion, *toVersion, *valuesPath, *outputPath, *registryURL); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runMigration(chartName, fromVersion, toVersion, valuesPath, outputPath, registryURL string) error {
	// Create migrator
	m := migrator.New(registryURL)

	// Perform migration
	result, err := m.Migrate(&migrator.MigrateRequest{
		ChartName:   chartName,
		FromVersion: fromVersion,
		ToVersion:   toVersion,
		ValuesPath:  valuesPath,
	})
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// Display summary
	fmt.Fprintf(os.Stderr, "\nMigration completed successfully!\n")
	fmt.Fprintf(os.Stderr, "Applied %d migration(s):\n", len(result.MigrationsUsed))
	for _, migration := range result.MigrationsUsed {
		fmt.Fprintf(os.Stderr, "  - %s\n", migration)
	}

	// Output result
	if outputPath != "" {
		// Write to file
		if err := os.WriteFile(outputPath, result.Migrated, 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "\nOutput written to: %s\n", outputPath)
	} else {
		// Write to stdout
		fmt.Fprintf(os.Stderr, "\n---\n")
		os.Stdout.Write(result.Migrated)
	}

	return nil
}
