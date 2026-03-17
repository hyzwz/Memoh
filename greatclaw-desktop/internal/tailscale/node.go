package tailscale

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"tailscale.com/tsnet"
)

// Node wraps a tsnet.Server for Tailscale network integration.
type Node struct {
	server *tsnet.Server
	logger *slog.Logger
}

// NewNode creates a new Tailscale node.
func NewNode(hostname string, log *slog.Logger) *Node {
	return &Node{
		server: &tsnet.Server{
			Hostname: hostname,
		},
		logger: log.With(slog.String("component", "tailscale")),
	}
}

// Start brings up the Tailscale node and returns its IP.
func (n *Node) Start(ctx context.Context) (string, error) {
	status, err := n.server.Up(ctx)
	if err != nil {
		return "", fmt.Errorf("tsnet up: %w", err)
	}

	if len(status.TailscaleIPs) == 0 {
		return "", fmt.Errorf("no Tailscale IPs assigned")
	}

	ip := status.TailscaleIPs[0].String()
	n.logger.Info("Tailscale node ready", slog.String("ip", ip), slog.String("hostname", n.server.Hostname))
	return ip, nil
}

// Listen creates a listener on the Tailscale network.
func (n *Node) Listen(network, addr string) (net.Listener, error) {
	return n.server.Listen(network, addr)
}

// Close shuts down the Tailscale node.
func (n *Node) Close() error {
	return n.server.Close()
}
