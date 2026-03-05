package enterprise

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/memohai/memoh/internal/db"
	"github.com/memohai/memoh/internal/db/sqlc"
	"github.com/memohai/memoh/internal/handlers"
)

// ModelRoutingService implements handlers.ModelRoutingServiceInterface.
type ModelRoutingService struct {
	queries *sqlc.Queries
	logger  *slog.Logger
}

func NewModelRoutingService(log *slog.Logger, queries *sqlc.Queries) *ModelRoutingService {
	return &ModelRoutingService{
		queries: queries,
		logger:  log.With(slog.String("service", "model-routing")),
	}
}

func (s *ModelRoutingService) ListRoutes(ctx context.Context, botID string) ([]handlers.ModelRouteDTO, error) {
	pgBotID, err := db.ParseUUID(botID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListModelRoutes(ctx, pgBotID)
	if err != nil {
		return nil, err
	}
	out := make([]handlers.ModelRouteDTO, len(rows))
	for i, r := range rows {
		out[i] = modelRouteToDTO(r)
	}
	return out, nil
}

func (s *ModelRoutingService) CreateRoute(ctx context.Context, botID string, req handlers.CreateModelRouteRequest) (*handlers.ModelRouteDTO, error) {
	pgBotID, err := db.ParseUUID(botID)
	if err != nil {
		return nil, err
	}

	var metadata []byte
	if req.Metadata != nil {
		var err error
		metadata, err = json.Marshal(req.Metadata)
		if err != nil {
			return nil, err
		}
	}

	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	route, err := s.queries.CreateModelRoute(ctx, sqlc.CreateModelRouteParams{
		BotID:          pgBotID,
		Name:           req.Name,
		ModelID:        db.ParseUUIDOrEmpty(req.ModelID),
		Priority:       int32(req.Priority),
		ComplexityTier: req.ComplexityTier,
		IsEnabled:      isEnabled,
		Metadata:       metadata,
	})
	if err != nil {
		return nil, err
	}
	dto := modelRouteToDTO(route)
	return &dto, nil
}

func (s *ModelRoutingService) DeleteRoute(ctx context.Context, botID, id string) error {
	pgID, err := db.ParseUUID(id)
	if err != nil {
		return err
	}

	// Verify route belongs to bot before deleting.
	route, err := s.queries.GetModelRouteByID(ctx, pgID)
	if err != nil {
		return err
	}
	pgBotID, err := db.ParseUUID(botID)
	if err != nil {
		return err
	}
	if route.BotID != pgBotID {
		return handlers.ErrRouteNotFound
	}

	return s.queries.DeleteModelRoute(ctx, pgID)
}

func modelRouteToDTO(r sqlc.BotModelRoute) handlers.ModelRouteDTO {
	var metadata map[string]any
	if r.Metadata != nil {
		_ = json.Unmarshal(r.Metadata, &metadata)
	}
	return handlers.ModelRouteDTO{
		ID:             uuidToString(r.ID),
		Name:           r.Name,
		ModelID:        uuidToString(r.ModelID),
		Priority:       int(r.Priority),
		ComplexityTier: r.ComplexityTier,
		IsEnabled:      r.IsEnabled,
		Metadata:       metadata,
	}
}
