package migration

import (
	"context"
	"fmt"
	"os"

	"github.com/kdwils/helmshift/pkg/config"
	"github.com/kdwils/helmshift/pkg/migration/formats"
	"gopkg.in/yaml.v3"
)

// Engine orchestrates the migration process
type Engine struct {
	config *config.MigrationConfig
	patch  Patch
}

// NewEngine creates a new migration engine
func NewEngine(configPath string) (*Engine, error) {
	cfg, err := config.LoadMigrationConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	patch, err := loadPatch(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to load patch: %w", err)
	}

	return &Engine{
		config: cfg,
		patch:  patch,
	}, nil
}

// Migrate performs the migration on the given values file
func (e *Engine) Migrate(ctx context.Context, valuesPath string) (*PatchResult, error) {
	// Read input values
	input, err := os.ReadFile(valuesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read values file: %w", err)
	}

	// Validate input is valid YAML
	if err := validateYAML(input); err != nil {
		return nil, fmt.Errorf("invalid input YAML: %w", err)
	}

	// Apply patch
	output, err := e.patch.Apply(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to apply patch: %w", err)
	}

	// Validate output is valid YAML
	if err := validateYAML(output); err != nil {
		return nil, fmt.Errorf("invalid output YAML: %w", err)
	}

	result := &PatchResult{
		Original: input,
		Output:   output,
		Format:   e.patch.Format(),
		Warnings: []string{},
	}

	// Run validation if configured
	if e.config.Validation != nil {
		if err := e.validateResult(result); err != nil {
			return nil, fmt.Errorf("validation failed: %w", err)
		}
	}

	return result, nil
}

// MigrateBytes performs migration on raw bytes (useful for testing/programmatic use)
func (e *Engine) MigrateBytes(ctx context.Context, input []byte) (*PatchResult, error) {
	// Validate input is valid YAML
	if err := validateYAML(input); err != nil {
		return nil, fmt.Errorf("invalid input YAML: %w", err)
	}

	// Apply patch
	output, err := e.patch.Apply(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to apply patch: %w", err)
	}

	// Validate output is valid YAML
	if err := validateYAML(output); err != nil {
		return nil, fmt.Errorf("invalid output YAML: %w", err)
	}

	result := &PatchResult{
		Original: input,
		Output:   output,
		Format:   e.patch.Format(),
		Warnings: []string{},
	}

	// Run validation if configured
	if e.config.Validation != nil {
		if err := e.validateResult(result); err != nil {
			return nil, fmt.Errorf("validation failed: %w", err)
		}
	}

	return result, nil
}

// loadPatch loads the appropriate patch implementation based on format
func loadPatch(cfg *config.MigrationConfig) (Patch, error) {
	switch cfg.Format {
	case "jsonpatch":
		return formats.NewJSONPatch(cfg.PatchFile)
	case "yq":
		return formats.NewYQPatch(cfg.PatchFile)
	case "script":
		return formats.NewScriptPatch(cfg.PatchFile)
	default:
		return nil, fmt.Errorf("unsupported patch format: %s", cfg.Format)
	}
}

// validateYAML checks if the data is valid YAML
func validateYAML(data []byte) error {
	var temp interface{}
	return yaml.Unmarshal(data, &temp)
}

// validateResult performs validation on the migration result
func (e *Engine) validateResult(result *PatchResult) error {
	var outputData map[string]interface{}
	if err := yaml.Unmarshal(result.Output, &outputData); err != nil {
		return fmt.Errorf("failed to parse output for validation: %w", err)
	}

	// Check required fields
	for _, field := range e.config.Validation.RequiredFields {
		if !hasField(outputData, field) {
			return fmt.Errorf("required field missing: %s", field)
		}
	}

	// Check forbidden fields
	for _, field := range e.config.Validation.ForbiddenFields {
		if hasField(outputData, field) {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("forbidden field present: %s", field))
		}
	}

	return nil
}

// hasField checks if a nested field exists in the data
func hasField(data map[string]interface{}, path string) bool {
	// Simple implementation - could be enhanced for nested paths
	_, exists := data[path]
	return exists
}
