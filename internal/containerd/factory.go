package containerd

import (
	"context"

	"github.com/containerd/containerd"
)

const (
	DefaultSocketPath = "/run/containerd/containerd.sock"
	DefaultNamespace  = "default"
)

type ClientFactory interface {
	New(ctx context.Context) (*containerd.Client, error)
}

type DefaultClientFactory struct {
	SocketPath string
}

func (f DefaultClientFactory) New(_ context.Context) (*containerd.Client, error) {
	socket := f.SocketPath
	if socket == "" {
		socket = DefaultSocketPath
	}
	return containerd.New(socket)
}
