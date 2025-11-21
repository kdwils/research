package operations

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Engine executes operations on YAML data
type Engine struct {
	data map[string]interface{}
}

// NewEngine creates a new transformation engine
func NewEngine(yamlData []byte) (*Engine, error) {
	var data map[string]interface{}
	if err := yaml.Unmarshal(yamlData, &data); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &Engine{data: data}, nil
}

// Apply applies a migration to the data
func (e *Engine) Apply(migration *Migration) error {
	for i, op := range migration.Operations {
		if err := e.applyOperation(op); err != nil {
			return fmt.Errorf("operation %d (%s) failed: %w", i, op.Op, err)
		}
	}
	return nil
}

// ToYAML returns the transformed data as YAML
func (e *Engine) ToYAML() ([]byte, error) {
	return yaml.Marshal(e.data)
}

// applyOperation applies a single operation
func (e *Engine) applyOperation(op Operation) error {
	switch op.Op {
	case OpSet:
		return e.opSet(op)
	case OpDelete:
		return e.opDelete(op)
	case OpMove:
		return e.opMove(op)
	case OpCopy:
		return e.opCopy(op)
	case OpRename:
		return e.opRename(op)
	default:
		return fmt.Errorf("unknown operation: %s", op.Op)
	}
}

// opSet sets a value at the given path
func (e *Engine) opSet(op Operation) error {
	path := parsePath(op.Path)
	value := op.Value

	// Apply transformation if specified
	if op.Transform != nil {
		transformed, err := applyTransform(value, op.Transform)
		if err != nil {
			return fmt.Errorf("transform failed: %w", err)
		}
		value = transformed
	}

	return e.setValue(path, value)
}

// opDelete deletes a value at the given path
func (e *Engine) opDelete(op Operation) error {
	path := parsePath(op.Path)
	return e.deleteValue(path)
}

// opMove moves a value from one path to another
func (e *Engine) opMove(op Operation) error {
	// Copy then delete
	if err := e.opCopy(op); err != nil {
		return err
	}
	return e.opDelete(Operation{Op: OpDelete, Path: op.From})
}

// opCopy copies a value from one path to another
func (e *Engine) opCopy(op Operation) error {
	fromPath := parsePath(op.From)
	toPath := parsePath(op.To)

	value, err := e.getValue(fromPath)
	if err != nil {
		return fmt.Errorf("failed to get source value: %w", err)
	}

	// Apply transformation if specified
	if op.Transform != nil {
		transformed, err := applyTransform(value, op.Transform)
		if err != nil {
			return fmt.Errorf("transform failed: %w", err)
		}
		value = transformed
	}

	return e.setValue(toPath, value)
}

// opRename renames a field (shorthand for move within same parent)
func (e *Engine) opRename(op Operation) error {
	return e.opMove(op)
}

// getValue retrieves a value at the given path
func (e *Engine) getValue(path []string) (interface{}, error) {
	current := interface{}(e.data)

	for i, key := range path {
		switch v := current.(type) {
		case map[string]interface{}:
			val, exists := v[key]
			if !exists {
				return nil, fmt.Errorf("path not found: %s", strings.Join(path[:i+1], "."))
			}
			current = val
		case map[interface{}]interface{}:
			val, exists := v[key]
			if !exists {
				return nil, fmt.Errorf("path not found: %s", strings.Join(path[:i+1], "."))
			}
			current = val
		default:
			return nil, fmt.Errorf("cannot traverse non-map at: %s", strings.Join(path[:i], "."))
		}
	}

	return current, nil
}

// setValue sets a value at the given path, creating intermediate maps as needed
func (e *Engine) setValue(path []string, value interface{}) error {
	if len(path) == 0 {
		return fmt.Errorf("empty path")
	}

	current := e.data

	// Traverse to parent, creating maps as needed
	for i := 0; i < len(path)-1; i++ {
		key := path[i]

		if next, exists := current[key]; exists {
			if nextMap, ok := next.(map[string]interface{}); ok {
				current = nextMap
			} else {
				return fmt.Errorf("cannot traverse non-map at: %s", strings.Join(path[:i+1], "."))
			}
		} else {
			// Create new map
			newMap := make(map[string]interface{})
			current[key] = newMap
			current = newMap
		}
	}

	// Set the final value
	current[path[len(path)-1]] = value
	return nil
}

// deleteValue deletes a value at the given path
func (e *Engine) deleteValue(path []string) error {
	if len(path) == 0 {
		return fmt.Errorf("empty path")
	}

	current := e.data

	// Traverse to parent
	for i := 0; i < len(path)-1; i++ {
		key := path[i]

		if next, exists := current[key]; exists {
			if nextMap, ok := next.(map[string]interface{}); ok {
				current = nextMap
			} else {
				return fmt.Errorf("cannot traverse non-map at: %s", strings.Join(path[:i+1], "."))
			}
		} else {
			// Path doesn't exist, nothing to delete
			return nil
		}
	}

	// Delete the final key
	delete(current, path[len(path)-1])
	return nil
}

// parsePath converts a dot-separated path to a slice
func parsePath(path string) []string {
	if path == "" {
		return []string{}
	}
	return strings.Split(path, ".")
}

// applyTransform applies a transformation to a value
func applyTransform(value interface{}, transform *Transform) (interface{}, error) {
	// Only string transformations for now
	strValue, ok := value.(string)
	if !ok {
		return value, nil // Non-string values pass through unchanged
	}

	switch transform.Type {
	case TransformStripPrefix:
		prefix, ok := transform.Params["prefix"].(string)
		if !ok {
			return nil, fmt.Errorf("stripPrefix requires 'prefix' parameter")
		}
		return strings.TrimPrefix(strValue, prefix), nil

	case TransformStripSuffix:
		suffix, ok := transform.Params["suffix"].(string)
		if !ok {
			return nil, fmt.Errorf("stripSuffix requires 'suffix' parameter")
		}
		return strings.TrimSuffix(strValue, suffix), nil

	case TransformReplace:
		old, okOld := transform.Params["old"].(string)
		new, okNew := transform.Params["new"].(string)
		if !okOld || !okNew {
			return nil, fmt.Errorf("replace requires 'old' and 'new' parameters")
		}
		return strings.ReplaceAll(strValue, old, new), nil

	case TransformToUpperCase:
		return strings.ToUpper(strValue), nil

	case TransformToLowerCase:
		return strings.ToLower(strValue), nil

	case TransformToCamelCase:
		return toCamelCase(strValue), nil

	case TransformToKebabCase:
		return toKebabCase(strValue), nil

	case TransformToSnakeCase:
		return toSnakeCase(strValue), nil

	default:
		return nil, fmt.Errorf("unknown transform type: %s", transform.Type)
	}
}

// toCamelCase converts kebab-case or snake_case to camelCase
func toCamelCase(s string) string {
	// Split on - or _
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_'
	})

	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}

	return strings.Join(parts, "")
}

// toKebabCase converts camelCase or snake_case to kebab-case
func toKebabCase(s string) string {
	// Replace underscores with hyphens
	s = strings.ReplaceAll(s, "_", "-")

	// Insert hyphen before uppercase letters
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('-')
		}
		result.WriteRune(r)
	}

	return strings.ToLower(result.String())
}

// toSnakeCase converts camelCase or kebab-case to snake_case
func toSnakeCase(s string) string {
	// Replace hyphens with underscores
	s = strings.ReplaceAll(s, "-", "_")

	// Insert underscore before uppercase letters
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}

	return strings.ToLower(result.String())
}
