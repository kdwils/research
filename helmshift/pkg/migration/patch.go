package migration

import (
	"context"
)

// Patch defines the interface that all patch formats must implement
type Patch interface {
	// Apply applies the patch to the input values and returns the result
	Apply(ctx context.Context, input []byte) ([]byte, error)

	// Validate checks if the patch is valid
	Validate() error

	// Format returns the format name (e.g., "jsonpatch", "yq", "script")
	Format() string
}

// PatchResult contains the result of applying a patch
type PatchResult struct {
	// Original input values
	Original []byte

	// Transformed output values
	Output []byte

	// Format used
	Format string

	// Any warnings generated during migration
	Warnings []string
}
