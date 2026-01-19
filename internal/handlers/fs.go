package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/namespaces"
	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/labstack/echo/v4"
	"github.com/pmezard/go-difflib/difflib"

	"github.com/memohai/memoh/internal/auth"
	"github.com/memohai/memoh/internal/config"
	ctr "github.com/memohai/memoh/internal/containerd"
	"github.com/memohai/memoh/internal/identity"
	"github.com/memohai/memoh/internal/mcp"
)

type FSHandler struct {
	service   ctr.Service
	manager   *mcp.Manager
	mcpConfig config.MCPConfig
	namespace string
}

type ErrorResponse struct {
	Message string `json:"message"`
}

type ReadResponse struct {
	Path     string    `json:"path"`
	Content  string    `json:"content"`
	Encoding string    `json:"encoding"`
	Size     int64     `json:"size"`
	Mode     uint32    `json:"mode"`
	ModTime  time.Time `json:"mod_time"`
}

type FileEntry struct {
	Path    string    `json:"path"`
	IsDir   bool      `json:"is_dir"`
	Size    int64     `json:"size"`
	Mode    uint32    `json:"mode"`
	ModTime time.Time `json:"mod_time"`
}

type ListResponse struct {
	Path    string      `json:"path"`
	Entries []FileEntry `json:"entries"`
}

type WriteAtomicRequest struct {
	Path     string     `json:"path"`
	Content  string     `json:"content"`
	Encoding string     `json:"encoding"`
	Mode     *uint32    `json:"mode,omitempty"`
	ModTime  *time.Time `json:"mtime,omitempty"`
}

type ApplyPatchRequest struct {
	Path  string `json:"path"`
	Patch string `json:"patch"`
}

type CommitResponse struct {
	ID         string    `json:"id"`
	Version    int       `json:"version"`
	SnapshotID string    `json:"snapshot_id"`
	CreatedAt  time.Time `json:"created_at"`
}

type DiffResponse struct {
	Path    string `json:"path"`
	Version int    `json:"version"`
	Diff    string `json:"diff"`
}

func NewFSHandler(service ctr.Service, manager *mcp.Manager, mcpConfig config.MCPConfig, namespace string) *FSHandler {
	if namespace == "" {
		namespace = config.DefaultNamespace
	}
	return &FSHandler{
		service:   service,
		manager:   manager,
		mcpConfig: mcpConfig,
		namespace: namespace,
	}
}

func (h *FSHandler) Register(e *echo.Echo) {
	group := e.Group("/fs")
	group.GET("/read", h.Read)
	group.GET("/list", h.List)
	group.PUT("/write_atomic", h.WriteAtomic)
	group.POST("/apply_patch", h.ApplyPatch)
	group.POST("/commit", h.Commit)
	group.GET("/diff", h.Diff)
}

// Read godoc
// @Summary Read file content
// @Description Read a file under the user data mount
// @Tags fs
// @Param path query string false "Path under data mount"
// @Success 200 {object} ReadResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /fs/read [get]
func (h *FSHandler) Read(c echo.Context) error {
	userID, err := h.requireUserID(c)
	if err != nil {
		return err
	}

	ctx := namespaces.WithNamespace(c.Request().Context(), h.namespace)
	mount, err := h.mountUser(ctx, userID)
	if err != nil {
		return err
	}
	defer mount.Unmount()

	containerPath, err := resolveContainerPath(h.dataMount(), c.QueryParam("path"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	hostPath, err := resolveHostPath(mount.Dir, containerPath)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	info, err := os.Stat(hostPath)
	if err != nil {
		if os.IsNotExist(err) {
			return echo.NewHTTPError(http.StatusNotFound, "file not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if info.IsDir() {
		return echo.NewHTTPError(http.StatusBadRequest, "path is a directory")
	}

	data, err := os.ReadFile(hostPath)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, ReadResponse{
		Path:     containerPath,
		Content:  base64.StdEncoding.EncodeToString(data),
		Encoding: "base64",
		Size:     info.Size(),
		Mode:     uint32(info.Mode().Perm()),
		ModTime:  info.ModTime(),
	})
}

// List godoc
// @Summary List directory contents
// @Description List files under the user data mount
// @Tags fs
// @Param path query string false "Path under data mount"
// @Param recursive query bool false "Recursive listing"
// @Success 200 {object} ListResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /fs/list [get]
func (h *FSHandler) List(c echo.Context) error {
	userID, err := h.requireUserID(c)
	if err != nil {
		return err
	}

	ctx := namespaces.WithNamespace(c.Request().Context(), h.namespace)
	mount, err := h.mountUser(ctx, userID)
	if err != nil {
		return err
	}
	defer mount.Unmount()

	containerPath, err := resolveContainerPath(h.dataMount(), c.QueryParam("path"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	hostPath, err := resolveHostPath(mount.Dir, containerPath)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	info, err := os.Stat(hostPath)
	if err != nil {
		if os.IsNotExist(err) {
			return echo.NewHTTPError(http.StatusNotFound, "path not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if !info.IsDir() {
		return echo.NewHTTPError(http.StatusBadRequest, "path is not a directory")
	}

	recursive := strings.EqualFold(c.QueryParam("recursive"), "true")
	entries := []FileEntry{}
	if recursive {
		err = filepath.WalkDir(hostPath, func(p string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if p == hostPath {
				return nil
			}
			entryInfo, err := d.Info()
			if err != nil {
				return err
			}
			containerEntry, err := containerPathForHost(mount.Dir, p)
			if err != nil {
				return err
			}
			entries = append(entries, FileEntry{
				Path:    containerEntry,
				IsDir:   d.IsDir(),
				Size:    entryInfo.Size(),
				Mode:    uint32(entryInfo.Mode().Perm()),
				ModTime: entryInfo.ModTime(),
			})
			return nil
		})
	} else {
		dirEntries, err := os.ReadDir(hostPath)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		for _, entry := range dirEntries {
			entryInfo, err := entry.Info()
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}
			entryPath := filepath.Join(hostPath, entry.Name())
			containerEntry, err := containerPathForHost(mount.Dir, entryPath)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}
			entries = append(entries, FileEntry{
				Path:    containerEntry,
				IsDir:   entry.IsDir(),
				Size:    entryInfo.Size(),
				Mode:    uint32(entryInfo.Mode().Perm()),
				ModTime: entryInfo.ModTime(),
			})
		}
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, ListResponse{
		Path:    containerPath,
		Entries: entries,
	})
}

// WriteAtomic godoc
// @Summary Write file atomically
// @Description Atomically replace a file under the user data mount
// @Tags fs
// @Param payload body WriteAtomicRequest true "Write payload"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /fs/write_atomic [put]
func (h *FSHandler) WriteAtomic(c echo.Context) error {
	userID, err := h.requireUserID(c)
	if err != nil {
		return err
	}

	var req WriteAtomicRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if req.Path == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "path is required")
	}

	ctx := namespaces.WithNamespace(c.Request().Context(), h.namespace)
	mount, err := h.mountUser(ctx, userID)
	if err != nil {
		return err
	}
	defer mount.Unmount()

	containerPath, err := resolveContainerPath(h.dataMount(), req.Path)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	hostPath, err := resolveHostPath(mount.Dir, containerPath)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	data, err := decodeContent(req.Content, req.Encoding)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	mode := os.FileMode(0o644)
	if req.Mode != nil {
		mode = os.FileMode(*req.Mode)
	}

	if err := writeFileAtomic(hostPath, data, mode, req.ModTime); err != nil {
		if os.IsNotExist(err) {
			return echo.NewHTTPError(http.StatusNotFound, "path not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}

// ApplyPatch godoc
// @Summary Apply unified diff patch
// @Description Apply a unified diff patch to a file under the user data mount
// @Tags fs
// @Param payload body ApplyPatchRequest true "Patch payload"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /fs/apply_patch [post]
func (h *FSHandler) ApplyPatch(c echo.Context) error {
	userID, err := h.requireUserID(c)
	if err != nil {
		return err
	}

	var req ApplyPatchRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if req.Path == "" || req.Patch == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "path and patch are required")
	}

	ctx := namespaces.WithNamespace(c.Request().Context(), h.namespace)
	mount, err := h.mountUser(ctx, userID)
	if err != nil {
		return err
	}
	defer mount.Unmount()

	containerPath, err := resolveContainerPath(h.dataMount(), req.Path)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	hostPath, err := resolveHostPath(mount.Dir, containerPath)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	orig, err := os.ReadFile(hostPath)
	if err != nil {
		if os.IsNotExist(err) {
			return echo.NewHTTPError(http.StatusNotFound, "file not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	updated, err := applyUnifiedPatch(string(orig), req.Patch)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	info, err := os.Stat(hostPath)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if err := writeFileAtomic(hostPath, []byte(updated), info.Mode().Perm(), nil); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}

// Commit godoc
// @Summary Commit a filesystem snapshot
// @Description Create a new version snapshot for the user container
// @Tags fs
// @Success 200 {object} CommitResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /fs/commit [post]
func (h *FSHandler) Commit(c echo.Context) error {
	userID, err := h.requireUserID(c)
	if err != nil {
		return err
	}

	if h.manager == nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "manager not configured")
	}

	ctx := namespaces.WithNamespace(c.Request().Context(), h.namespace)
	if err := h.ensureUserContainer(ctx, userID); err != nil {
		return err
	}

	info, err := h.manager.CreateVersion(ctx, userID)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return echo.NewHTTPError(http.StatusNotFound, "container not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, CommitResponse{
		ID:         info.ID,
		Version:    info.Version,
		SnapshotID: info.SnapshotID,
		CreatedAt:  info.CreatedAt,
	})
}

// Diff godoc
// @Summary Diff against a version snapshot
// @Description Produce a unified diff between a version snapshot and current data
// @Tags fs
// @Param path query string false "Path under data mount"
// @Param version query int true "Version number"
// @Success 200 {object} DiffResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /fs/diff [get]
func (h *FSHandler) Diff(c echo.Context) error {
	userID, err := h.requireUserID(c)
	if err != nil {
		return err
	}
	if h.manager == nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "manager not configured")
	}

	versionStr := c.QueryParam("version")
	if versionStr == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "version is required")
	}
	version, err := strconv.Atoi(versionStr)
	if err != nil || version <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid version")
	}

	containerPath, err := resolveContainerPath(h.dataMount(), c.QueryParam("path"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	ctx := namespaces.WithNamespace(c.Request().Context(), h.namespace)
	mount, err := h.mountUser(ctx, userID)
	if err != nil {
		return err
	}
	defer mount.Unmount()

	versionSnapshotID, err := h.manager.VersionSnapshotID(ctx, userID, version)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return echo.NewHTTPError(http.StatusNotFound, "version not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	versionDir, versionCleanup, err := ctr.MountSnapshot(ctx, h.service, mount.Info.Snapshotter, versionSnapshotID)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return echo.NewHTTPError(http.StatusNotFound, "snapshot not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	defer versionCleanup()

	currentHostPath, err := resolveHostPath(mount.Dir, containerPath)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	versionHostPath, err := resolveHostPath(versionDir, containerPath)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	currentContent, err := readFileOrEmpty(currentHostPath)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	versionContent, err := readFileOrEmpty(versionHostPath)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	diffText, err := unifiedDiff(containerPath, versionContent, currentContent)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, DiffResponse{
		Path:    containerPath,
		Version: version,
		Diff:    diffText,
	})
}

func (h *FSHandler) dataMount() string {
	if h.mcpConfig.DataMount == "" {
		return config.DefaultDataMount
	}
	return h.mcpConfig.DataMount
}

func (h *FSHandler) requireUserID(c echo.Context) (string, error) {
	userID, err := auth.UserIDFromContext(c)
	if err != nil {
		return "", err
	}
	if err := identity.ValidateUserID(userID); err != nil {
		return "", echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return userID, nil
}

func (h *FSHandler) mountUser(ctx context.Context, userID string) (*ctr.MountedSnapshot, error) {
	containerID := mcp.ContainerPrefix + userID
	mount, err := ctr.MountContainerSnapshot(ctx, h.service, containerID)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return nil, echo.NewHTTPError(http.StatusNotFound, "container not found")
		}
		return nil, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if label, ok := mount.Info.Labels[mcp.UserLabelKey]; !ok || label != userID {
		_ = mount.Unmount()
		return nil, echo.NewHTTPError(http.StatusForbidden, "user mismatch")
	}
	return mount, nil
}

func (h *FSHandler) ensureUserContainer(ctx context.Context, userID string) error {
	containerID := mcp.ContainerPrefix + userID
	container, err := h.service.GetContainer(ctx, containerID)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return echo.NewHTTPError(http.StatusNotFound, "container not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	info, err := container.Info(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if label, ok := info.Labels[mcp.UserLabelKey]; !ok || label != userID {
		return echo.NewHTTPError(http.StatusForbidden, "user mismatch")
	}
	return nil
}

func resolveContainerPath(dataMount, requestPath string) (string, error) {
	mountPath := path.Clean(dataMount)
	if mountPath == "." || !strings.HasPrefix(mountPath, "/") {
		return "", fmt.Errorf("data mount must be absolute")
	}

	if requestPath == "" {
		return mountPath, nil
	}

	reqClean := path.Clean(requestPath)
	if path.IsAbs(reqClean) {
		if !pathWithin(reqClean, mountPath) {
			return "", fmt.Errorf("path outside data mount")
		}
		return reqClean, nil
	}

	return path.Join(mountPath, reqClean), nil
}

func pathWithin(target, base string) bool {
	if base == "/" {
		return strings.HasPrefix(target, "/")
	}
	if target == base {
		return true
	}
	if strings.HasPrefix(target, base) {
		return len(target) > len(base) && target[len(base)] == '/'
	}
	return false
}

func resolveHostPath(mountDir, containerPath string) (string, error) {
	rel := strings.TrimPrefix(containerPath, "/")
	return securejoin.SecureJoin(mountDir, rel)
}

func containerPathForHost(mountDir, hostPath string) (string, error) {
	rel, err := filepath.Rel(mountDir, hostPath)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path escapes mount")
	}
	return "/" + filepath.ToSlash(rel), nil
}

func decodeContent(content, encoding string) ([]byte, error) {
	switch strings.ToLower(encoding) {
	case "", "plain":
		return []byte(content), nil
	case "base64":
		return base64.StdEncoding.DecodeString(content)
	default:
		return nil, fmt.Errorf("unsupported encoding")
	}
}

func writeFileAtomic(targetPath string, data []byte, mode os.FileMode, modTime *time.Time) error {
	dir := filepath.Dir(targetPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := io.Copy(tmp, bytes.NewReader(data)); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if modTime != nil {
		if err := os.Chtimes(tmpName, *modTime, *modTime); err != nil {
			return err
		}
	}
	if err := os.Rename(tmpName, targetPath); err != nil {
		return err
	}
	if modTime != nil {
		_ = os.Chtimes(targetPath, *modTime, *modTime)
	}
	return nil
}

func applyUnifiedPatch(original, patch string) (string, error) {
	lines := strings.Split(original, "\n")
	out := make([]string, 0, len(lines))
	index := 0
	patchLines := strings.Split(patch, "\n")
	hunksApplied := 0

	for i := 0; i < len(patchLines); i++ {
		line := patchLines[i]
		if !strings.HasPrefix(line, "@@") {
			continue
		}

		origStart, err := parseUnifiedHunkHeader(line)
		if err != nil {
			return "", err
		}
		origStart--
		if origStart < 0 {
			origStart = 0
		}
		if origStart > len(lines) {
			return "", fmt.Errorf("patch out of range")
		}

		out = append(out, lines[index:origStart]...)
		index = origStart
		hunksApplied++

		for i+1 < len(patchLines) {
			next := patchLines[i+1]
			if strings.HasPrefix(next, "@@") {
				break
			}
			i++

			if next == "" {
				if i == len(patchLines)-1 {
					break
				}
				return "", fmt.Errorf("invalid patch line")
			}
			if next[0] == '\\' {
				continue
			}
			if len(next) < 1 {
				return "", fmt.Errorf("invalid patch line")
			}
			op := next[0]
			text := next[1:]
			switch op {
			case ' ':
				if index >= len(lines) || lines[index] != text {
					return "", fmt.Errorf("patch context mismatch")
				}
				out = append(out, text)
				index++
			case '-':
				if index >= len(lines) || lines[index] != text {
					return "", fmt.Errorf("patch delete mismatch")
				}
				index++
			case '+':
				out = append(out, text)
			default:
				return "", fmt.Errorf("invalid patch operation")
			}
		}
	}
	if hunksApplied == 0 {
		return "", fmt.Errorf("patch contains no hunks")
	}

	out = append(out, lines[index:]...)
	return strings.Join(out, "\n"), nil
}

func parseUnifiedHunkHeader(header string) (int, error) {
	trimmed := strings.TrimPrefix(header, "@@")
	trimmed = strings.TrimSpace(trimmed)
	if !strings.HasPrefix(trimmed, "-") {
		return 0, fmt.Errorf("invalid hunk header")
	}
	parts := strings.SplitN(trimmed, " ", 2)
	if len(parts) < 2 {
		return 0, fmt.Errorf("invalid hunk header")
	}

	origPart := strings.TrimPrefix(parts[0], "-")
	origFields := strings.SplitN(origPart, ",", 2)
	origStart, err := strconv.Atoi(origFields[0])
	if err != nil {
		return 0, fmt.Errorf("invalid hunk header")
	}
	return origStart, nil
}

func readFileOrEmpty(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

func unifiedDiff(containerPath, oldContent, newContent string) (string, error) {
	diffText, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        strings.Split(oldContent, "\n"),
		B:        strings.Split(newContent, "\n"),
		FromFile: "a" + containerPath,
		ToFile:   "b" + containerPath,
		Context:  3,
	})
	if err != nil {
		return "", err
	}
	return diffText, nil
}
