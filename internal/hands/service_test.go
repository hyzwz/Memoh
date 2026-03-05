package hands

import (
	"context"
	"encoding/json"
	"testing"
)

// fakeStore implements the Store interface for unit testing.
type fakeStore struct {
	hands          map[string]*HandRecord // keyed by ID
	handsByBotName map[string]*HandRecord // keyed by botID+name
	nextID         int
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		hands:          make(map[string]*HandRecord),
		handsByBotName: make(map[string]*HandRecord),
	}
}

func (f *fakeStore) CreateHand(_ context.Context, p CreateHandInput) (*HandRecord, error) {
	f.nextID++
	id := idStr(f.nextID)
	r := &HandRecord{
		ID:               id,
		BotID:            p.BotID,
		Name:             p.Name,
		Description:      p.Description,
		Content:          p.Content,
		Type:             p.Type,
		SchedulePattern:  p.SchedulePattern,
		ScheduleTimezone: p.ScheduleTimezone,
		ScheduleMaxCalls: p.ScheduleMaxCalls,
		TriggersJSON:     p.TriggersJSON,
		AllowedToolsJSON: p.AllowedToolsJSON,
		ModelConfigJSON:  p.ModelConfigJSON,
		OutputConfigJSON: p.OutputConfigJSON,
		PermissionsJSON:  p.PermissionsJSON,
		IsEnabled:        p.IsEnabled,
		MetadataJSON:     p.MetadataJSON,
	}
	f.hands[id] = r
	f.handsByBotName[p.BotID+"/"+p.Name] = r
	return r, nil
}

func (f *fakeStore) GetHandByID(_ context.Context, id string) (*HandRecord, error) {
	r, ok := f.hands[id]
	if !ok {
		return nil, ErrHandNotFound
	}
	return r, nil
}

func (f *fakeStore) GetHandByName(_ context.Context, botID, name string) (*HandRecord, error) {
	r, ok := f.handsByBotName[botID+"/"+name]
	if !ok {
		return nil, ErrHandNotFound
	}
	return r, nil
}

func (f *fakeStore) ListHands(_ context.Context, botID string) ([]*HandRecord, error) {
	var result []*HandRecord
	for _, r := range f.hands {
		if r.BotID == botID {
			result = append(result, r)
		}
	}
	return result, nil
}

func (f *fakeStore) DeleteHand(_ context.Context, id string) error {
	r, ok := f.hands[id]
	if !ok {
		return ErrHandNotFound
	}
	delete(f.handsByBotName, r.BotID+"/"+r.Name)
	delete(f.hands, id)
	return nil
}

func idStr(n int) string {
	return "hand-" + string(rune('0'+n))
}

func TestServiceCreateFromMarkdown(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	md := `---
name: daily-report
description: Daily team report
type: hand
schedule:
  pattern: "0 0 18 * * 1-5"
  timezone: "Asia/Shanghai"
triggers:
  - event: message_keyword
    match: "generate report"
tools:
  - web_fetch
  - send
model:
  complexity_tier: medium
output:
  channels: ["wecom"]
  target: department_group
permissions:
  require_approval: false
  max_tokens_per_run: 50000
---

You are a daily report generator.
`

	hand, err := svc.CreateFromMarkdown(context.Background(), "bot-1", md)
	if err != nil {
		t.Fatalf("CreateFromMarkdown: %v", err)
	}
	if hand.Name != "daily-report" {
		t.Fatalf("name = %q", hand.Name)
	}
	if hand.Type != "hand" {
		t.Fatalf("type = %q", hand.Type)
	}

	// Verify stored record has serialized JSON.
	rec, err := store.GetHandByName(context.Background(), "bot-1", "daily-report")
	if err != nil {
		t.Fatal(err)
	}
	if rec.SchedulePattern != "0 0 18 * * 1-5" {
		t.Fatalf("schedule_pattern = %q", rec.SchedulePattern)
	}

	var triggers []TriggerDef
	if err := json.Unmarshal([]byte(rec.TriggersJSON), &triggers); err != nil {
		t.Fatal(err)
	}
	if len(triggers) != 1 || triggers[0].Match != "generate report" {
		t.Fatalf("triggers = %v", triggers)
	}

	var tools []string
	if err := json.Unmarshal([]byte(rec.AllowedToolsJSON), &tools); err != nil {
		t.Fatal(err)
	}
	if len(tools) != 2 || tools[0] != "web_fetch" {
		t.Fatalf("tools = %v", tools)
	}
}

func TestServiceGet(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	_, err := svc.CreateFromMarkdown(context.Background(), "bot-1", `---
name: test-hand
type: hand
---

Test content.
`)
	if err != nil {
		t.Fatal(err)
	}

	// Get by name
	hand, err := svc.GetByName(context.Background(), "bot-1", "test-hand")
	if err != nil {
		t.Fatalf("GetByName: %v", err)
	}
	if hand.Name != "test-hand" {
		t.Fatalf("name = %q", hand.Name)
	}
	if hand.Content != "Test content." {
		t.Fatalf("content = %q", hand.Content)
	}
}

func TestServiceList(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	ctx := context.Background()
	_, _ = svc.CreateFromMarkdown(ctx, "bot-1", "---\nname: h1\ntype: hand\n---\nContent 1")
	_, _ = svc.CreateFromMarkdown(ctx, "bot-1", "---\nname: h2\ntype: hand\n---\nContent 2")
	_, _ = svc.CreateFromMarkdown(ctx, "bot-2", "---\nname: h3\ntype: hand\n---\nContent 3")

	hands, err := svc.List(ctx, "bot-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(hands) != 2 {
		t.Fatalf("expected 2 hands for bot-1, got %d", len(hands))
	}
}

func TestServiceDelete(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	ctx := context.Background()
	hand, _ := svc.CreateFromMarkdown(ctx, "bot-1", "---\nname: to-delete\ntype: hand\n---\nDelete me")

	err := svc.Delete(ctx, hand.ID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.GetByName(ctx, "bot-1", "to-delete")
	if err != ErrHandNotFound {
		t.Fatalf("expected ErrHandNotFound, got %v", err)
	}
}

func TestServiceGetNotFound(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	_, err := svc.GetByName(context.Background(), "bot-1", "nonexistent")
	if err != ErrHandNotFound {
		t.Fatalf("expected ErrHandNotFound, got %v", err)
	}
}
