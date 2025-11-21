package formats

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"gopkg.in/yaml.v3"
)

// JSONPatch implements RFC 6902 JSON Patch format
type JSONPatch struct {
	patchData []byte
	patch     jsonpatch.Patch
}

// NewJSONPatch creates a new JSON Patch from a file
func NewJSONPatch(patchFile string) (*JSONPatch, error) {
	data, err := os.ReadFile(patchFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read patch file: %w", err)
	}

	// Try to parse as YAML first (more user-friendly)
	var patchOps interface{}
	if err := yaml.Unmarshal(data, &patchOps); err != nil {
		return nil, fmt.Errorf("failed to parse patch file as YAML: %w", err)
	}

	// Convert to JSON for the jsonpatch library
	jsonData, err := json.Marshal(patchOps)
	if err != nil {
		return nil, fmt.Errorf("failed to convert patch to JSON: %w", err)
	}

	// Parse the JSON Patch
	patch, err := jsonpatch.DecodePatch(jsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JSON patch: %w", err)
	}

	return &JSONPatch{
		patchData: data,
		patch:     patch,
	}, nil
}

// Apply applies the JSON Patch to the input values
func (p *JSONPatch) Apply(ctx context.Context, input []byte) ([]byte, error) {
	// Convert YAML to JSON
	var inputData interface{}
	if err := yaml.Unmarshal(input, &inputData); err != nil {
		return nil, fmt.Errorf("failed to parse input YAML: %w", err)
	}

	inputJSON, err := json.Marshal(inputData)
	if err != nil {
		return nil, fmt.Errorf("failed to convert input to JSON: %w", err)
	}

	// Apply the patch
	modifiedJSON, err := p.patch.Apply(inputJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to apply patch: %w", err)
	}

	// Convert back to YAML
	var outputData interface{}
	if err := json.Unmarshal(modifiedJSON, &outputData); err != nil {
		return nil, fmt.Errorf("failed to parse patched JSON: %w", err)
	}

	outputYAML, err := yaml.Marshal(outputData)
	if err != nil {
		return nil, fmt.Errorf("failed to convert output to YAML: %w", err)
	}

	return outputYAML, nil
}

// Validate checks if the patch is valid
func (p *JSONPatch) Validate() error {
	// The patch is validated during construction
	return nil
}

// Format returns the format name
func (p *JSONPatch) Format() string {
	return "jsonpatch"
}
