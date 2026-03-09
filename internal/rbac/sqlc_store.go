package rbac

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/memohai/memoh/internal/db"
	"github.com/memohai/memoh/internal/db/sqlc"
)

// SQLCStore adapts generated queries to the RBAC resolver store contract.
type SQLCStore struct {
	queries *sqlc.Queries
}

func NewSQLCStore(queries *sqlc.Queries) *SQLCStore {
	return &SQLCStore{queries: queries}
}

func (s *SQLCStore) GetUserContext(ctx context.Context, userID string) (*UserContext, error) {
	if s == nil || s.queries == nil {
		return nil, errors.New("rbac queries not configured")
	}

	pgUserID, err := db.ParseUUID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	user, err := s.queries.GetUserByID(ctx, pgUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	departments, err := s.queries.ListUserDepartments(ctx, pgUserID)
	if err != nil {
		return nil, err
	}

	memberships, err := s.queries.ListBotMembershipsByUser(ctx, pgUserID)
	if err != nil {
		return nil, err
	}

	userCtx := &UserContext{
		UserID:        userID,
		SystemRole:    Role(strings.TrimSpace(user.Role)),
		Departments:   make([]DepartmentMembership, 0, len(departments)),
		BotMembership: make(map[string]BotRole, len(memberships)),
	}

	for _, item := range departments {
		userCtx.Departments = append(userCtx.Departments, DepartmentMembership{
			DepartmentID: item.ID.String(),
			Role:         strings.TrimSpace(item.MemberRole),
		})
	}

	for _, item := range memberships {
		userCtx.BotMembership[item.BotID.String()] = BotRole(strings.TrimSpace(item.Role))
	}

	return userCtx, nil
}

func (s *SQLCStore) GetBotDepartmentAccess(ctx context.Context, botID string) ([]string, error) {
	if s == nil || s.queries == nil {
		return nil, errors.New("rbac queries not configured")
	}

	pgBotID, err := db.ParseUUID(botID)
	if err != nil {
		return nil, err
	}

	rows, err := s.queries.ListBotDepartments(ctx, pgBotID)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(rows))
	for _, item := range rows {
		ids = append(ids, item.ID.String())
	}
	return ids, nil
}

func (s *SQLCStore) GetBotPermissions(ctx context.Context, botID string) ([]Permission, error) {
	if s == nil || s.queries == nil {
		return nil, errors.New("rbac queries not configured")
	}

	pgBotID, err := db.ParseUUID(botID)
	if err != nil {
		return nil, err
	}

	rows, err := s.queries.ListBotPermissions(ctx, pgBotID)
	if err != nil {
		return nil, err
	}

	perms := make([]Permission, 0, len(rows))
	for _, item := range rows {
		perms = append(perms, Permission{
			BotID:    item.BotID.String(),
			Role:     BotRole(strings.TrimSpace(item.Role)),
			Resource: Resource(strings.TrimSpace(item.Resource)),
			Action:   Action(strings.TrimSpace(item.Action)),
			Allowed:  item.Allowed,
		})
	}
	return perms, nil
}
