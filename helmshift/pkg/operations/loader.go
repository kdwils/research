package operations

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// LoadMigration parses a migration from YAML data
func LoadMigration(data []byte) (*Migration, error) {
	var migration Migration
	if err := yaml.Unmarshal(data, &migration); err != nil {
		return nil, fmt.Errorf("failed to parse migration: %w", err)
	}

	// Validate migration
	if err := migration.Validate(); err != nil {
		return nil, fmt.Errorf("invalid migration: %w", err)
	}

	return &migration, nil
}

// Validate checks if the migration is valid
func (m *Migration) Validate() error {
	if m.ChartName == "" {
		return fmt.Errorf("chartName is required")
	}

	if m.FromVersion == "" {
		return fmt.Errorf("fromVersion is required")
	}

	if m.ToVersion == "" {
		return fmt.Errorf("toVersion is required")
	}

	if len(m.Operations) == 0 {
		return fmt.Errorf("at least one operation is required")
	}

	// Validate each operation
	for i, op := range m.Operations {
		if err := op.Validate(); err != nil {
			return fmt.Errorf("operation %d invalid: %w", i, err)
		}
	}

	return nil
}

// Validate checks if an operation is valid
func (op *Operation) Validate() error {
	if op.Op == "" {
		return fmt.Errorf("op is required")
	}

	switch op.Op {
	case OpSet:
		if op.Path == "" {
			return fmt.Errorf("set operation requires path")
		}
		if op.Value == nil {
			return fmt.Errorf("set operation requires value")
		}

	case OpDelete:
		if op.Path == "" {
			return fmt.Errorf("delete operation requires path")
		}

	case OpMove, OpCopy, OpRename:
		if op.From == "" {
			return fmt.Errorf("%s operation requires from", op.Op)
		}
		if op.To == "" {
			return fmt.Errorf("%s operation requires to", op.Op)
		}

	default:
		return fmt.Errorf("unknown operation: %s", op.Op)
	}

	return nil
}
