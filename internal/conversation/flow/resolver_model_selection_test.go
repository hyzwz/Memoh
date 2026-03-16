package flow

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/memohai/memoh/internal/conversation"
	"github.com/memohai/memoh/internal/models"
)

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

func TestModelOverrideHook_Applied(t *testing.T) {
	t.Parallel()
	r := &Resolver{
		logger: slog.Default(),
	}
	r.SetModelOverride(func(_ context.Context, _, _ string, _ conversation.ChatRequest) (string, error) {
		return "overridden-model", nil
	})
	if r.modelOverride == nil {
		t.Fatal("expected modelOverride hook to be set")
	}
	got, err := r.modelOverride(context.Background(), "bot-1", "original-model", conversation.ChatRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "overridden-model" {
		t.Fatalf("expected overridden-model, got %s", got)
	}
}

func TestModelOverrideHook_ReceivesRequest(t *testing.T) {
	t.Parallel()
	r := &Resolver{
		logger: slog.Default(),
	}
	r.SetModelOverride(func(_ context.Context, _, _ string, req conversation.ChatRequest) (string, error) {
		tier := ResolvePreferredModelRouteTier(req)
		if tier == "complex" {
			return "heavy-model", nil
		}
		return "light-model", nil
	})
	got, err := r.modelOverride(context.Background(), "bot-1", "model-1",
		conversation.ChatRequest{Query: strings.Repeat("detailed architecture analysis\n", 12)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "heavy-model" {
		t.Fatalf("expected heavy-model for complex query, got %s", got)
	}
}

func TestModelOverrideHook_ErrorPassthrough(t *testing.T) {
	t.Parallel()
	r := &Resolver{
		logger: slog.Default(),
	}
	r.SetModelOverride(func(_ context.Context, _, _ string, _ conversation.ChatRequest) (string, error) {
		return "", fmt.Errorf("routing error")
	})
	_, err := r.modelOverride(context.Background(), "bot-1", "model-1", conversation.ChatRequest{})
	if err == nil {
		t.Fatal("expected error from model override hook")
	}
}
