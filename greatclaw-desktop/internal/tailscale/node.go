package tailscale

import (
	"context"
	"fmt"
	"log/slog"
	"net"
)

// Node wraps a Tailscale network node.
// TODO: Re-enable tsnet integration once Wails CLI supports go 1.26+ toolchain.
// Current limitation: Wails v2.11 binding generator fails with tsnet's type system requirements.
type Node struct {
	hostname string
	logger   *slog.Logger
}

// NewNode creates a new Tailscale node.
func NewNode(hostname string, log *slog.Logger) *Node {
	return &Node{
		hostname: hostname,
		logger:   log.With(slog.String("component", "tailscale")),
	}
}

// Start brings up the Tailscale node and returns its IP.
// Placeholder: returns localhost until tsnet integration is re-enabled.
func (n *Node) Start(ctx context.Context) (string, error) {
	n.logger.Warn("tsnet disabled — using localhost for File API")
	return "127.0.0.1", nil
}

// Listen creates a TCP listener.
// Placeholder: uses standard net.Listen until tsnet integration is re-enabled.
func (n *Node) Listen(network, addr string) (net.Listener, error) {
	ln, err := net.Listen(network, addr)
	if err != nil {
		return nil, fmt.Errorf("listen %s: %w", addr, err)
	}
	n.logger.Info("File API listening (local mode)", slog.String("addr", ln.Addr().String()))
	return ln, nil
}

// Close shuts down the node.
func (n *Node) Close() error {
	return nil
}
