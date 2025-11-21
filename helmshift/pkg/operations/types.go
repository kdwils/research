package operations

// Migration defines a set of operations to transform values
type Migration struct {
	Version     string      `yaml:"version"`
	ChartName   string      `yaml:"chartName"`
	FromVersion string      `yaml:"fromVersion"`
	ToVersion   string      `yaml:"toVersion"`
	Description string      `yaml:"description,omitempty"`
	Operations  []Operation `yaml:"operations"`
}

// Operation represents a single transformation operation
type Operation struct {
	// Operation type: set, delete, move, copy, rename
	Op string `yaml:"op"`

	// Path for set/delete operations
	Path string `yaml:"path,omitempty"`

	// Value for set operations
	Value interface{} `yaml:"value,omitempty"`

	// Source path for move/copy operations
	From string `yaml:"from,omitempty"`

	// Destination path for move/copy/rename operations
	To string `yaml:"to,omitempty"`

	// Optional transformations
	Transform *Transform `yaml:"transform,omitempty"`
}

// Transform defines optional value transformations
type Transform struct {
	// Type of transformation
	Type string `yaml:"type"`

	// Parameters for transformation
	Params map[string]interface{} `yaml:"params,omitempty"`
}

// Supported operation types
const (
	OpSet    = "set"    // Set a value at path
	OpDelete = "delete" // Delete a path
	OpMove   = "move"   // Move from -> to (copy + delete from)
	OpCopy   = "copy"   // Copy from -> to
	OpRename = "rename" // Rename field (within same parent)
)

// Supported transformations
const (
	TransformStripPrefix  = "stripPrefix"
	TransformStripSuffix  = "stripSuffix"
	TransformReplace      = "replace"
	TransformToUpperCase  = "toUpperCase"
	TransformToLowerCase  = "toLowerCase"
	TransformToCamelCase  = "toCamelCase"
	TransformToKebabCase  = "toKebabCase"
	TransformToSnakeCase  = "toSnakeCase"
)
