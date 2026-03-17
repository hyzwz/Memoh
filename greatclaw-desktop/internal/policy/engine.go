package policy

import (
	"os"
	"path/filepath"
	"strings"
)

// Engine implements the share policy: deny by default, allow listed roots.
type Engine struct {
	roots      []string
	neverShare []string
}

// NewEngine creates a policy engine with the given share roots.
func NewEngine(roots []string) *Engine {
	return &Engine{
		roots:      roots,
		neverShare: DefaultNeverShare(),
	}
}

// IsAllowed checks if a file path is permitted by the policy.
func (e *Engine) IsAllowed(absPath string) bool {
	// Check against never-share patterns
	name := filepath.Base(absPath)
	for _, pattern := range e.neverShare {
		if matched, _ := filepath.Match(pattern, name); matched {
			return false
		}
		// Also check parent directories
		if strings.Contains(absPath, string(filepath.Separator)+pattern+string(filepath.Separator)) {
			return false
		}
	}

	// Check the path is within a share root
	for _, root := range e.roots {
		absRoot, _ := filepath.Abs(root)
		if strings.HasPrefix(absPath, absRoot) {
			return true
		}
	}
	return false
}

// ShareRoots returns the configured share roots.
func (e *Engine) ShareRoots() []string {
	expanded := make([]string, 0, len(e.roots))
	for _, r := range e.roots {
		if strings.HasPrefix(r, "~/") {
			home, _ := os.UserHomeDir()
			r = filepath.Join(home, r[2:])
		}
		abs, err := filepath.Abs(r)
		if err == nil {
			expanded = append(expanded, abs)
		}
	}
	return expanded
}

// SetRoots updates the share roots.
func (e *Engine) SetRoots(roots []string) {
	e.roots = roots
}
