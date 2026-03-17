package fileapi

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ListRequest is the request body for POST /api/v1/files/list
type ListRequest struct {
	Path      string `json:"path"`
	Recursive bool   `json:"recursive"`
	Pattern   string `json:"pattern"`
}

// FileEntry represents a single file/directory in a listing.
type FileEntry struct {
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	Size       int64     `json:"size"`
	IsDir      bool      `json:"is_dir"`
	ModifiedAt time.Time `json:"modified_at"`
	Extension  string    `json:"extension,omitempty"`
}

// ListResponse is the response for POST /api/v1/files/list
type ListResponse struct {
	Files []FileEntry `json:"files"`
}

// ReadRequest is the request body for POST /api/v1/files/read
type ReadRequest struct {
	Path   string `json:"path"`
	Offset int    `json:"offset,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

// ReadResponse is the response for POST /api/v1/files/read
type ReadResponse struct {
	Path       string    `json:"path"`
	Content    string    `json:"content"`
	Encoding   string    `json:"encoding"`
	Size       int64     `json:"size"`
	TotalLines int       `json:"total_lines"`
	Truncated  bool      `json:"truncated"`
	ModifiedAt time.Time `json:"modified_at"`
}

// SearchRequest is the request body for POST /api/v1/files/search
type SearchRequest struct {
	Query      string `json:"query"`
	Path       string `json:"path,omitempty"`
	Glob       string `json:"glob,omitempty"`
	MaxResults int    `json:"max_results,omitempty"`
}

// SearchMatch represents a single search hit.
type SearchMatch struct {
	Line    int    `json:"line"`
	Content string `json:"content"`
}

// SearchResult represents search results for one file.
type SearchResult struct {
	Path    string        `json:"path"`
	Matches []SearchMatch `json:"matches"`
}

// SearchResponse is the response for POST /api/v1/files/search
type SearchResponse struct {
	Results      []SearchResult `json:"results"`
	TotalMatches int            `json:"total_matches"`
	Truncated    bool           `json:"truncated"`
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	var req ListRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	absPath, err := s.resolvePath(req.Path)
	if err != nil {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	var entries []FileEntry
	walkFn := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if !req.Recursive && path != absPath {
			if d.IsDir() {
				// For non-recursive, only include immediate children
				rel, _ := filepath.Rel(absPath, path)
				if strings.Contains(rel, string(filepath.Separator)) {
					return fs.SkipDir
				}
			}
		}
		if path == absPath {
			return nil // skip root itself
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		relPath, _ := filepath.Rel(s.findShareRoot(absPath), path)
		entry := FileEntry{
			Name:       d.Name(),
			Path:       relPath,
			Size:       info.Size(),
			IsDir:      d.IsDir(),
			ModifiedAt: info.ModTime(),
		}
		if !d.IsDir() {
			entry.Extension = filepath.Ext(d.Name())
		}

		if req.Pattern != "" {
			matched, _ := filepath.Match(req.Pattern, d.Name())
			if !matched {
				return nil
			}
		}

		entries = append(entries, entry)
		return nil
	}

	filepath.WalkDir(absPath, walkFn)

	writeJSON(w, ListResponse{Files: entries})
}

func (s *Server) handleRead(w http.ResponseWriter, r *http.Request) {
	var req ReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	absPath, err := s.resolvePath(req.Path)
	if err != nil {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	info, err := os.Stat(absPath)
	if err != nil {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}
	if info.IsDir() {
		writeError(w, http.StatusBadRequest, "path is a directory")
		return
	}

	// Read file with size limit (10MB)
	const maxSize = 10 * 1024 * 1024
	if info.Size() > maxSize {
		writeError(w, http.StatusBadRequest, "file too large")
		return
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read file")
		return
	}

	content := string(data)
	lines := strings.Split(content, "\n")
	totalLines := len(lines)
	truncated := false

	// Apply offset/limit
	if req.Offset > 0 && req.Offset < totalLines {
		lines = lines[req.Offset:]
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 500
	}
	if len(lines) > limit {
		lines = lines[:limit]
		truncated = true
	}

	writeJSON(w, ReadResponse{
		Path:       req.Path,
		Content:    strings.Join(lines, "\n"),
		Encoding:   "utf-8",
		Size:       info.Size(),
		TotalLines: totalLines,
		Truncated:  truncated,
		ModifiedAt: info.ModTime(),
	})
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Query == "" {
		writeError(w, http.StatusBadRequest, "query is required")
		return
	}
	if req.MaxResults <= 0 {
		req.MaxResults = 20
	}

	// Search within share roots
	searchPath := ""
	if req.Path != "" {
		resolved, err := s.resolvePath(req.Path)
		if err != nil {
			writeError(w, http.StatusForbidden, "access denied")
			return
		}
		searchPath = resolved
	}

	var results []SearchResult
	totalMatches := 0

	roots := s.policy.ShareRoots()
	if searchPath != "" {
		roots = []string{searchPath}
	}

	for _, root := range roots {
		filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if totalMatches >= req.MaxResults {
				return fs.SkipAll
			}
			// Check policy for each file in search
			if !s.policy.IsAllowed(path) {
				return nil
			}
			if req.Glob != "" {
				matched, _ := filepath.Match(req.Glob, d.Name())
				if !matched {
					return nil
				}
			}

			info, _ := d.Info()
			if info != nil && info.Size() > 5*1024*1024 {
				return nil // skip large files
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			lines := strings.Split(string(data), "\n")
			var matches []SearchMatch
			for i, line := range lines {
				if strings.Contains(line, req.Query) {
					matches = append(matches, SearchMatch{
						Line:    i + 1,
						Content: line,
					})
					totalMatches++
					if totalMatches >= req.MaxResults {
						break
					}
				}
			}

			if len(matches) > 0 {
				relPath, _ := filepath.Rel(root, path)
				results = append(results, SearchResult{
					Path:    relPath,
					Matches: matches,
				})
			}
			return nil
		})
	}

	writeJSON(w, SearchResponse{
		Results:      results,
		TotalMatches: totalMatches,
		Truncated:    totalMatches >= req.MaxResults,
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{"status": "ok"})
}
