package fileapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// resolvePath validates and resolves a relative path against share roots.
// It prevents path traversal, symlink escapes, and ADS attacks.
func (s *Server) resolvePath(relPath string) (string, error) {
	if relPath == "" {
		relPath = "."
	}

	// Block obvious traversal attempts
	cleaned := filepath.Clean(relPath)
	if strings.Contains(cleaned, "..") {
		return "", fmt.Errorf("path traversal detected")
	}

	// Block Windows ADS
	if strings.Contains(relPath, ":") && len(relPath) > 2 {
		return "", fmt.Errorf("alternate data streams not allowed")
	}

	// Try each share root
	for _, root := range s.policy.ShareRoots() {
		candidate := filepath.Join(root, cleaned)

		// Resolve symlinks and check the real path stays within root
		real, err := filepath.EvalSymlinks(candidate)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", err
		}

		absRoot, _ := filepath.Abs(root)
		// Ensure path separator follows root to prevent /share-root-evil matching /share-root
		if real != absRoot && !strings.HasPrefix(real, absRoot+string(filepath.Separator)) {
			return "", fmt.Errorf("symlink escape detected")
		}

		if !s.policy.IsAllowed(real) {
			return "", fmt.Errorf("access denied by policy")
		}

		return real, nil
	}

	return "", fmt.Errorf("path not found in any share root")
}

// findShareRoot returns the share root that contains the given absolute path.
func (s *Server) findShareRoot(absPath string) string {
	for _, root := range s.policy.ShareRoots() {
		absRoot, _ := filepath.Abs(root)
		if absPath == absRoot || strings.HasPrefix(absPath, absRoot+string(filepath.Separator)) {
			return absRoot
		}
	}
	return ""
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
