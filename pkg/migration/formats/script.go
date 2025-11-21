package formats

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ScriptPatch implements script-based transformations
type ScriptPatch struct {
	scriptPath string
	interpreter string
}

// NewScriptPatch creates a new script-based patch
func NewScriptPatch(scriptPath string) (*ScriptPatch, error) {
	// Check if script exists
	if _, err := os.Stat(scriptPath); err != nil {
		return nil, fmt.Errorf("script file not found: %w", err)
	}

	// Determine interpreter based on file extension
	interpreter := determineInterpreter(scriptPath)

	// Check if script is executable (or interpreter is available)
	if interpreter != "" {
		if _, err := exec.LookPath(interpreter); err != nil {
			return nil, fmt.Errorf("interpreter %s not found: %w", interpreter, err)
		}
	}

	return &ScriptPatch{
		scriptPath: scriptPath,
		interpreter: interpreter,
	}, nil
}

// Apply executes the script with input on stdin and captures stdout
func (p *ScriptPatch) Apply(ctx context.Context, input []byte) ([]byte, error) {
	var cmd *exec.Cmd

	if p.interpreter != "" {
		cmd = exec.CommandContext(ctx, p.interpreter, p.scriptPath)
	} else {
		cmd = exec.CommandContext(ctx, p.scriptPath)
	}

	// Provide input values via stdin
	cmd.Stdin = bytes.NewReader(input)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute the script
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("script execution failed: %w\nstderr: %s", err, stderr.String())
	}

	// Return the transformed output
	output := stdout.Bytes()
	if len(output) == 0 {
		return nil, fmt.Errorf("script produced no output")
	}

	return output, nil
}

// Validate checks if the script is executable
func (p *ScriptPatch) Validate() error {
	info, err := os.Stat(p.scriptPath)
	if err != nil {
		return fmt.Errorf("cannot access script: %w", err)
	}

	// Check if it's a regular file
	if !info.Mode().IsRegular() {
		return fmt.Errorf("script path is not a regular file")
	}

	// If no interpreter, check if executable
	if p.interpreter == "" {
		if info.Mode().Perm()&0111 == 0 {
			return fmt.Errorf("script is not executable and no interpreter specified")
		}
	}

	return nil
}

// Format returns the format name
func (p *ScriptPatch) Format() string {
	return "script"
}

// determineInterpreter returns the appropriate interpreter based on file extension
func determineInterpreter(scriptPath string) string {
	ext := filepath.Ext(scriptPath)
	switch ext {
	case ".sh", ".bash":
		return "bash"
	case ".py", ".python":
		return "python3"
	case ".rb", ".ruby":
		return "ruby"
	case ".js":
		return "node"
	case ".pl":
		return "perl"
	default:
		return "" // Assume script has shebang or is executable
	}
}
