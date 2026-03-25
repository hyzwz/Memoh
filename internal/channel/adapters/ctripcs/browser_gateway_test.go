package ctripcs

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestBrowserGatewayClientBuildsContextActionRequest(t *testing.T) {
	t.Parallel()

	var sawMethod string
	var sawPath string
	var sawContentType string
	var sawBody []byte

	rt := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		sawMethod = r.Method
		sawPath = r.URL.Path
		sawContentType = r.Header.Get("Content-Type")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		sawBody = append([]byte(nil), body...)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"success":true,"data":{"ok":true}}`)),
			Header:     make(http.Header),
		}, nil
	})

	client := &browserGatewayClient{
		baseURL: "http://browser-gateway.test",
		client:  &http.Client{Transport: rt},
	}

	resp, err := client.doAction(context.Background(), "ctx-123", browserGatewayActionRequest{
		Action: "evaluate",
		Script: "return window.__MEMOH__",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if string(resp) != `{"success":true,"data":{"ok":true}}` {
		t.Fatalf("unexpected response: %s", string(resp))
	}
	if sawMethod != http.MethodPost {
		t.Fatalf("unexpected method: %s", sawMethod)
	}
	if sawPath != "/context/ctx-123/action" {
		t.Fatalf("unexpected path: %s", sawPath)
	}
	if sawContentType != "application/json" {
		t.Fatalf("unexpected content type: %s", sawContentType)
	}
	if string(sawBody) != `{"action":"evaluate","script":"return window.__MEMOH__"}` {
		t.Fatalf("unexpected request body: %s", string(sawBody))
	}
}

func TestBrowserGatewayClientChecksContextExists(t *testing.T) {
	t.Parallel()

	var sawMethod string
	var sawPath string

	rt := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		sawMethod = r.Method
		sawPath = r.URL.Path
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"exists":true}`)),
			Header:     make(http.Header),
		}, nil
	})

	client := &browserGatewayClient{
		baseURL: "http://browser-gateway.test",
		client:  &http.Client{Transport: rt},
	}

	exists, err := client.Exists(context.Background(), "ctx-123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !exists {
		t.Fatal("expected context to exist")
	}
	if sawMethod != http.MethodGet {
		t.Fatalf("unexpected method: %s", sawMethod)
	}
	if sawPath != "/context/ctx-123/exists" {
		t.Fatalf("unexpected path: %s", sawPath)
	}
}

func TestBrowserGatewayClientEvaluatesSnapshotScript(t *testing.T) {
	t.Parallel()

	rt := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"success":true,"data":{"result":{"page_url":"https://m.ctrip.com/customer-service/inbox"}}}`)),
			Header:     make(http.Header),
		}, nil
	})

	client := &browserGatewayClient{
		baseURL: "http://browser-gateway.test",
		client:  &http.Client{Transport: rt},
	}

	raw, err := client.Evaluate(context.Background(), "ctx-123", "return window.__MEMOH__")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if string(raw) != `{"page_url":"https://m.ctrip.com/customer-service/inbox"}` {
		t.Fatalf("unexpected evaluate result: %s", string(raw))
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
