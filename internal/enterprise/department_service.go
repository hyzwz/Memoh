package enterprise

import (
	"context"
	"log/slog"

	"github.com/memohai/memoh/internal/db"
	"github.com/memohai/memoh/internal/db/sqlc"
	"github.com/memohai/memoh/internal/handlers"
)

// DepartmentService implements handlers.DepartmentServiceInterface.
type DepartmentService struct {
	queries *sqlc.Queries
	logger  *slog.Logger
}

func NewDepartmentService(log *slog.Logger, queries *sqlc.Queries) *DepartmentService {
	return &DepartmentService{
		queries: queries,
		logger:  log.With(slog.String("service", "departments")),
	}
}

func (s *DepartmentService) ListDepartments(ctx context.Context, botID string) ([]handlers.DepartmentDTO, error) {
	pgBotID, err := db.ParseUUID(botID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListBotDepartments(ctx, pgBotID)
	if err != nil {
		return nil, err
	}
	out := make([]handlers.DepartmentDTO, len(rows))
	for i, r := range rows {
		out[i] = departmentToDTO(r)
	}
	return out, nil
}

func (s *DepartmentService) CreateDepartment(ctx context.Context, botID string, req handlers.CreateDepartmentRequest) (*handlers.DepartmentDTO, error) {
	pgBotID, err := db.ParseUUID(botID)
	if err != nil {
		return nil, err
	}

	dept, err := s.queries.CreateDepartment(ctx, sqlc.CreateDepartmentParams{
		Name:        req.Name,
		Description: req.Description,
		ParentID:    db.ParseUUIDOrEmpty(req.ParentID),
		Metadata:    nil,
	})
	if err != nil {
		return nil, err
	}

	// Associate department with bot.
	if err := s.queries.SetBotDepartmentAccess(ctx, sqlc.SetBotDepartmentAccessParams{
		BotID:        pgBotID,
		DepartmentID: dept.ID,
	}); err != nil {
		return nil, err
	}

	dto := departmentToDTO(dept)
	return &dto, nil
}

func departmentToDTO(d sqlc.Department) handlers.DepartmentDTO {
	return handlers.DepartmentDTO{
		ID:          uuidToString(d.ID),
		Name:        d.Name,
		Description: d.Description,
		ParentID:    uuidToString(d.ParentID),
	}
}
