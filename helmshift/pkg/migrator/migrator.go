package migrator

import (
	"fmt"
	"os"

	"github.com/kdwils/helmshift/pkg/operations"
	"github.com/kdwils/helmshift/pkg/registry"
)

// Migrator orchestrates the migration process
type Migrator struct {
	registryClient *registry.RegistryClient
}

// New creates a new migrator
func New(registryURL string) *Migrator {
	return &Migrator{
		registryClient: registry.NewRegistryClient(registryURL),
	}
}

// MigrateRequest contains parameters for a migration
type MigrateRequest struct {
	ChartName   string
	FromVersion string
	ToVersion   string
	ValuesPath  string
}

// MigrateResult contains the result of a migration
type MigrateResult struct {
	Original       []byte
	Migrated       []byte
	MigrationsUsed []string
}

// Migrate performs a migration
func (m *Migrator) Migrate(req *MigrateRequest) (*MigrateResult, error) {
	// 1. Fetch migration index
	fmt.Fprintf(os.Stderr, "Fetching migration index for %s...\n", req.ChartName)
	index, err := m.registryClient.FetchIndex(req.ChartName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch migration index: %w", err)
	}

	// 2. Find migration path
	fmt.Fprintf(os.Stderr, "Finding migration path from %s to %s...\n", req.FromVersion, req.ToVersion)
	path, err := index.FindMigrationPath(req.FromVersion, req.ToVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to find migration path: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Migration path: ")
	for i, step := range path {
		if i > 0 {
			fmt.Fprintf(os.Stderr, " → ")
		}
		fmt.Fprintf(os.Stderr, "%s", step.ToVersion)
	}
	fmt.Fprintf(os.Stderr, "\n")

	// 3. Read original values
	originalValues, err := os.ReadFile(req.ValuesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read values file: %w", err)
	}

	// 4. Apply migrations sequentially
	currentValues := originalValues
	migrationsUsed := []string{}

	for i, step := range path {
		fmt.Fprintf(os.Stderr, "Applying migration %d/%d: %s → %s...\n",
			i+1, len(path), step.FromVersion, step.ToVersion)

		// Fetch migration file
		migrationData, err := m.registryClient.FetchMigration(step.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch migration %s: %w", step.URL, err)
		}

		// Parse migration
		migration, err := operations.LoadMigration(migrationData)
		if err != nil {
			return nil, fmt.Errorf("failed to load migration: %w", err)
		}

		// Apply migration
		engine, err := operations.NewEngine(currentValues)
		if err != nil {
			return nil, fmt.Errorf("failed to create engine: %w", err)
		}

		if err := engine.Apply(migration); err != nil {
			return nil, fmt.Errorf("failed to apply migration %s: %w", step.URL, err)
		}

		// Get result for next iteration
		currentValues, err = engine.ToYAML()
		if err != nil {
			return nil, fmt.Errorf("failed to serialize result: %w", err)
		}

		migrationsUsed = append(migrationsUsed, fmt.Sprintf("%s → %s", step.FromVersion, step.ToVersion))
	}

	return &MigrateResult{
		Original:       originalValues,
		Migrated:       currentValues,
		MigrationsUsed: migrationsUsed,
	}, nil
}

// MigrateBytes performs migration on raw bytes (for testing)
func (m *Migrator) MigrateBytes(chartName, fromVersion, toVersion string, values []byte) ([]byte, error) {
	// 1. Fetch migration index
	index, err := m.registryClient.FetchIndex(chartName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch migration index: %w", err)
	}

	// 2. Find migration path
	path, err := index.FindMigrationPath(fromVersion, toVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to find migration path: %w", err)
	}

	// 3. Apply migrations sequentially
	currentValues := values

	for _, step := range path {
		// Fetch migration file
		migrationData, err := m.registryClient.FetchMigration(step.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch migration %s: %w", step.URL, err)
		}

		// Parse migration
		migration, err := operations.LoadMigration(migrationData)
		if err != nil {
			return nil, fmt.Errorf("failed to load migration: %w", err)
		}

		// Apply migration
		engine, err := operations.NewEngine(currentValues)
		if err != nil {
			return nil, fmt.Errorf("failed to create engine: %w", err)
		}

		if err := engine.Apply(migration); err != nil {
			return nil, fmt.Errorf("failed to apply migration: %w", err)
		}

		// Get result for next iteration
		currentValues, err = engine.ToYAML()
		if err != nil {
			return nil, fmt.Errorf("failed to serialize result: %w", err)
		}
	}

	return currentValues, nil
}
