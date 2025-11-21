package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"gopkg.in/yaml.v3"
)

// RegistryClient fetches migrations from a remote registry
type RegistryClient struct {
	baseURL    string
	httpClient *http.Client
}

// MigrationIndex represents the index of available migrations for a chart
type MigrationIndex struct {
	ChartName  string              `yaml:"chartName"`
	Migrations []MigrationMetadata `yaml:"migrations"`
}

// MigrationMetadata describes a single migration
type MigrationMetadata struct {
	FromVersion string `yaml:"fromVersion"`
	ToVersion   string `yaml:"toVersion"`
	URL         string `yaml:"url"` // Relative or absolute URL to migration file
	Description string `yaml:"description,omitempty"`
}

// NewRegistryClient creates a new registry client
func NewRegistryClient(baseURL string) *RegistryClient {
	return &RegistryClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchIndex fetches the migration index for a chart
func (c *RegistryClient) FetchIndex(chartName string) (*MigrationIndex, error) {
	url := fmt.Sprintf("%s/%s/index.yaml", c.baseURL, chartName)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch index from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch index: HTTP %d from %s", resp.StatusCode, url)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read index response: %w", err)
	}

	var index MigrationIndex
	if err := yaml.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("failed to parse index: %w", err)
	}

	return &index, nil
}

// FetchMigration fetches a specific migration file
func (c *RegistryClient) FetchMigration(url string) ([]byte, error) {
	// If URL is relative, prepend base URL
	if url[0] == '/' || (len(url) > 0 && url[0] != 'h') {
		url = c.baseURL + url
	}

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch migration from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch migration: HTTP %d from %s", resp.StatusCode, url)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read migration response: %w", err)
	}

	return data, nil
}

// FindMigrationPath finds the chain of migrations needed to go from -> to
func (idx *MigrationIndex) FindMigrationPath(fromVersion, toVersion string) ([]MigrationMetadata, error) {
	// For PoC, implement simple sequential matching
	// Production would use proper semver matching and graph traversal

	var path []MigrationMetadata
	currentVersion := fromVersion

	for currentVersion != toVersion {
		found := false
		for _, migration := range idx.Migrations {
			if matchesVersion(currentVersion, migration.FromVersion) {
				path = append(path, migration)
				currentVersion = migration.ToVersion
				found = true
				break
			}
		}

		if !found {
			return nil, fmt.Errorf("no migration path found from %s to %s", fromVersion, toVersion)
		}

		// Prevent infinite loops
		if len(path) > 20 {
			return nil, fmt.Errorf("migration path too long (possible cycle)")
		}
	}

	return path, nil
}

// matchesVersion checks if a version matches a version pattern
// Simplified for PoC - production would use proper semver
func matchesVersion(version, pattern string) bool {
	// Exact match
	if version == pattern {
		return true
	}

	// Pattern with wildcard (e.g., "0.1.x" matches "0.1.5")
	if len(pattern) > 2 && pattern[len(pattern)-1] == 'x' {
		prefix := pattern[:len(pattern)-1]
		return len(version) >= len(prefix) && version[:len(prefix)] == prefix
	}

	return false
}

// FormatAsJSON returns the index as JSON (useful for debugging)
func (idx *MigrationIndex) FormatAsJSON() (string, error) {
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
