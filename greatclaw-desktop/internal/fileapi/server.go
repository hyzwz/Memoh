package fileapi

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// Server is the File API server that listens on the Tailscale network.
type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
	policy     PolicyChecker
	apiToken   string
}

// PolicyChecker validates file access against the share policy.
type PolicyChecker interface {
	IsAllowed(path string) bool
	ShareRoots() []string
}

// NewServer creates a new File API server.
func NewServer(log *slog.Logger, policy PolicyChecker, apiToken string) *Server {
	s := &Server{
		logger: log.With(slog.String("component", "fileapi")),
		policy: policy,
		apiToken: apiToken,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/files/list", s.withAuth(s.handleList))
	mux.HandleFunc("POST /api/v1/files/read", s.withAuth(s.handleRead))
	mux.HandleFunc("POST /api/v1/files/search", s.withAuth(s.handleSearch))
	mux.HandleFunc("GET /api/v1/health", s.handleHealth)

	s.httpServer = &http.Server{
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	return s
}

// Serve starts the HTTP server on the given listener.
func (s *Server) Serve(ln net.Listener) error {
	s.logger.Info("File API server starting", slog.String("addr", ln.Addr().String()))
	return s.httpServer.Serve(ln)
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
