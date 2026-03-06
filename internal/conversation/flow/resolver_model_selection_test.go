package flow

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/memohai/memoh/internal/models"
)

// --- Model routing test fakes ---

type fakeModelRouter struct {
	routes []ModelRouteEntry
	err    error
}

func (f *fakeModelRouter) ListEnabledRoutes(_ context.Context, _ string) ([]ModelRouteEntry, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.routes, nil
}

func TestMatchesModelReference_ModelID(t *testing.T) {
	t.Parallel()

	model := models.GetResponse{
		ID:      "a55f0d2d-1547-49a0-b085-ec4ab778f4b8",
		ModelID: "gpt-4o",
	}

	if !matchesModelReference(model, "gpt-4o") {
		t.Fatal("expected model slug to match")
	}
}

func TestMatchesModelReference_UUID(t *testing.T) {
	t.Parallel()

	model := models.GetResponse{
		ID:      "a55f0d2d-1547-49a0-b085-ec4ab778f4b8",
		ModelID: "gpt-4o",
	}

	if !matchesModelReference(model, "a55f0d2d-1547-49a0-b085-ec4ab778f4b8") {
		t.Fatal("expected model UUID to match")
	}
}

func TestMatchesModelReference_NoMatch(t *testing.T) {
	t.Parallel()

	model := models.GetResponse{
		ID:      "a55f0d2d-1547-49a0-b085-ec4ab778f4b8",
		ModelID: "gpt-4o",
	}

	if matchesModelReference(model, "gpt-4.1") {
		t.Fatal("expected non-matching model reference to fail")
	}
}

func TestMatchesModelReference_TrimmedInput(t *testing.T) {
	t.Parallel()

	model := models.GetResponse{
		ID:      "a55f0d2d-1547-49a0-b085-ec4ab778f4b8",
		ModelID: "gpt-4o",
	}

	if !matchesModelReference(model, "  gpt-4o  ") {
		t.Fatal("expected trimmed model slug to match")
	}
}

func TestResolveModelRoute_ReturnsHighestPriority(t *testing.T) {
	t.Parallel()
	r := &Resolver{
		logger: slog.Default(),
		modelRouter: &fakeModelRouter{
			routes: []ModelRouteEntry{
				{ModelID: "model-high", Priority: 100, IsEnabled: true},
				{ModelID: "model-low", Priority: 10, IsEnabled: true},
			},
		},
	}
	modelID, ok := r.resolveModelRoute(context.Background(), "bot-1")
	if !ok {
		t.Fatal("expected route to be found")
	}
	if modelID != "model-high" {
		t.Fatalf("expected model-high, got %s", modelID)
	}
}

func TestResolveModelRoute_NoRoutes(t *testing.T) {
	t.Parallel()
	r := &Resolver{
		logger: slog.Default(),
		modelRouter: &fakeModelRouter{
			routes: []ModelRouteEntry{},
		},
	}
	_, ok := r.resolveModelRoute(context.Background(), "bot-1")
	if ok {
		t.Fatal("expected no route found")
	}
}

func TestResolveModelRoute_ErrorFallsThrough(t *testing.T) {
	t.Parallel()
	r := &Resolver{
		logger: slog.Default(),
		modelRouter: &fakeModelRouter{
			err: fmt.Errorf("db error"),
		},
	}
	_, ok := r.resolveModelRoute(context.Background(), "bot-1")
	if ok {
		t.Fatal("expected no route on error (graceful degradation)")
	}
}

func TestResolveModelRoute_SkipsDisabled(t *testing.T) {
	t.Parallel()
	r := &Resolver{
		logger: slog.Default(),
		modelRouter: &fakeModelRouter{
			routes: []ModelRouteEntry{
				{ModelID: "model-disabled", Priority: 100, IsEnabled: false},
				{ModelID: "model-enabled", Priority: 50, IsEnabled: true},
			},
		},
	}
	modelID, ok := r.resolveModelRoute(context.Background(), "bot-1")
	if !ok {
		t.Fatal("expected route to be found")
	}
	if modelID != "model-enabled" {
		t.Fatalf("expected model-enabled, got %s", modelID)
	}
}
