package wecombot

import (
	"context"
	"testing"
	"time"
)

func TestDispatchInboundFrameDoesNotBlockReadLoop(t *testing.T) {
	t.Parallel()

	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})

	client := newWSClient(nil, Config{}, func(context.Context, wsFrame) error {
		close(started)
		<-release
		close(done)
		return nil
	})

	returned := make(chan struct{})
	go func() {
		client.dispatchInboundFrame(context.Background(), wsFrame{Body: []byte(`{"msgtype":"text"}`)})
		close(returned)
	}()

	select {
	case <-started:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("handler did not start")
	}

	select {
	case <-returned:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("dispatch blocked while handler was still running")
	}

	select {
	case <-done:
		t.Fatal("handler completed before release")
	default:
	}

	close(release)

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("handler did not complete after release")
	}
}
