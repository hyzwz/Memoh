package hands

import (
	"context"
	"encoding/json"
	"errors"
)

var ErrHandNotFound = errors.New("hand not found")

// HandRecord is the flat database representation of a hand.
type HandRecord struct {
	ID               string
	BotID            string
	Name             string
	Description      string
	Content          string
	Type             string
	SchedulePattern  string
	ScheduleTimezone string
	ScheduleMaxCalls *int
	TriggersJSON     string
	AllowedToolsJSON string
	ModelConfigJSON  string
	OutputConfigJSON string
	PermissionsJSON  string
	IsEnabled        bool
	MetadataJSON     string
}

// CreateHandInput contains the fields needed to create a hand record.
type CreateHandInput struct {
	BotID            string
	Name             string
	Description      string
	Content          string
	Type             string
	SchedulePattern  string
	ScheduleTimezone string
	ScheduleMaxCalls *int
	TriggersJSON     string
	AllowedToolsJSON string
	ModelConfigJSON  string
	OutputConfigJSON string
	PermissionsJSON  string
	IsEnabled        bool
	MetadataJSON     string
}

// Store abstracts database access for hands.
type Store interface {
	CreateHand(ctx context.Context, input CreateHandInput) (*HandRecord, error)
	GetHandByID(ctx context.Context, id string) (*HandRecord, error)
	GetHandByName(ctx context.Context, botID, name string) (*HandRecord, error)
	ListHands(ctx context.Context, botID string) ([]*HandRecord, error)
	DeleteHand(ctx context.Context, id string) error
}

// Service provides hand lifecycle operations.
type Service struct {
	store Store
}

// NewService creates a new hand service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// CreateFromMarkdown parses a HAND.md and persists the hand.
func (s *Service) CreateFromMarkdown(ctx context.Context, botID, markdown string) (*HandRecord, error) {
	h := ParseHandMD(markdown, "")
	if h.Name == "" {
		return nil, errors.New("hand name is required")
	}
	if h.Content == "" {
		return nil, errors.New("hand content is required")
	}
	return s.store.CreateHand(ctx, handToInput(botID, h))
}

// GetByName retrieves a hand by bot ID and name.
func (s *Service) GetByName(ctx context.Context, botID, name string) (*HandRecord, error) {
	return s.store.GetHandByName(ctx, botID, name)
}

// GetByID retrieves a hand by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*HandRecord, error) {
	return s.store.GetHandByID(ctx, id)
}

// List returns all hands for a bot.
func (s *Service) List(ctx context.Context, botID string) ([]*HandRecord, error) {
	return s.store.ListHands(ctx, botID)
}

// Delete removes a hand by ID.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.DeleteHand(ctx, id)
}

// RecordToHand converts a HandRecord back to a Hand domain object.
func RecordToHand(r *HandRecord) Hand {
	h := Hand{
		Name:        r.Name,
		Description: r.Description,
		Type:        r.Type,
		Content:     r.Content,
	}

	if r.SchedulePattern != "" {
		tz := r.ScheduleTimezone
		if tz == "" {
			tz = "UTC"
		}
		h.Schedule = &ScheduleConfig{
			Pattern:  r.SchedulePattern,
			Timezone: tz,
			MaxCalls: r.ScheduleMaxCalls,
		}
	}

	if r.TriggersJSON != "" && r.TriggersJSON != "[]" {
		_ = json.Unmarshal([]byte(r.TriggersJSON), &h.Triggers)
	}
	if r.AllowedToolsJSON != "" && r.AllowedToolsJSON != "[]" {
		_ = json.Unmarshal([]byte(r.AllowedToolsJSON), &h.AllowedTools)
	}
	if r.ModelConfigJSON != "" && r.ModelConfigJSON != "{}" {
		_ = json.Unmarshal([]byte(r.ModelConfigJSON), &h.ModelConfig)
	}
	if r.OutputConfigJSON != "" && r.OutputConfigJSON != "{}" {
		_ = json.Unmarshal([]byte(r.OutputConfigJSON), &h.OutputConfig)
	}
	if r.PermissionsJSON != "" && r.PermissionsJSON != "{}" {
		_ = json.Unmarshal([]byte(r.PermissionsJSON), &h.Permissions)
	}
	if r.MetadataJSON != "" && r.MetadataJSON != "{}" {
		_ = json.Unmarshal([]byte(r.MetadataJSON), &h.Metadata)
	}

	return h
}

func handToInput(botID string, h Hand) CreateHandInput {
	input := CreateHandInput{
		BotID:       botID,
		Name:        h.Name,
		Description: h.Description,
		Content:     h.Content,
		Type:        h.Type,
		IsEnabled:   true,
	}

	if h.Schedule != nil {
		input.SchedulePattern = h.Schedule.Pattern
		input.ScheduleTimezone = h.Schedule.Timezone
		input.ScheduleMaxCalls = h.Schedule.MaxCalls
	}

	input.TriggersJSON = mustJSON(h.Triggers)
	input.AllowedToolsJSON = mustJSON(h.AllowedTools)
	input.ModelConfigJSON = mustJSON(h.ModelConfig)
	input.OutputConfigJSON = mustJSON(h.OutputConfig)
	input.PermissionsJSON = mustJSON(h.Permissions)
	input.MetadataJSON = mustJSON(h.Metadata)

	return input
}

func mustJSON(v any) string {
	if v == nil {
		return "{}"
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}
