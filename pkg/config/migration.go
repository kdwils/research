package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// MigrationConfig defines the metadata for a migration
type MigrationConfig struct {
	// Version information
	FromVersion string `yaml:"fromVersion"`
	ToVersion   string `yaml:"toVersion"`

	// Patch format: "jsonpatch", "yq", or "script"
	Format string `yaml:"format"`

	// Description of what this migration does
	Description string `yaml:"description"`

	// Path to the patch file (relative to config file)
	PatchFile string `yaml:"patchFile"`

	// Optional: validation rules
	Validation *ValidationConfig `yaml:"validation,omitempty"`
}

// ValidationConfig defines pre/post migration validation
type ValidationConfig struct {
	// Required fields in the output
	RequiredFields []string `yaml:"requiredFields,omitempty"`

	// Fields that should not exist in output
	ForbiddenFields []string `yaml:"forbiddenFields,omitempty"`
}

// LoadMigrationConfig loads a migration configuration from a YAML file
func LoadMigrationConfig(path string) (*MigrationConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config MigrationConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Resolve patch file path relative to config directory
	if config.PatchFile != "" && !filepath.IsAbs(config.PatchFile) {
		configDir := filepath.Dir(path)
		config.PatchFile = filepath.Join(configDir, config.PatchFile)
	}

	return &config, nil
}
