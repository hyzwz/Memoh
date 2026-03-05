package sqlc_test

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/memohai/memoh/internal/db/sqlc"
)

func setupTestDB(t *testing.T) (*sqlc.Queries, *pgxpool.Pool, func()) {
	t.Helper()
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("skip integration test: TEST_POSTGRES_DSN is not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("skip: cannot connect: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("skip: ping failed: %v", err)
	}
	queries := sqlc.New(pool)
	return queries, pool, func() { pool.Close() }
}

func numericFromFloat(f float64) pgtype.Numeric {
	// Use string conversion to avoid floating point issues.
	s := fmt.Sprintf("%.2f", f)
	var n pgtype.Numeric
	n.ScanScientific(s)
	return n
}

func numericFromString(s string) pgtype.Numeric {
	var n pgtype.Numeric
	bi := new(big.Int)
	// Simple: parse as integer * 10^(-scale)
	// For "0.80" → Int=80, Exp=-2
	// For "100.00" → Int=10000, Exp=-2
	switch s {
	case "0.80":
		bi.SetInt64(80)
		n.Int = bi
		n.Exp = -2
		n.Valid = true
	case "100.00":
		bi.SetInt64(10000)
		n.Int = bi
		n.Exp = -2
		n.Valid = true
	default:
		n.ScanScientific(s)
	}
	return n
}

// createTestUser inserts a minimal user for FK references.
func createTestUser(t *testing.T, q *sqlc.Queries) pgtype.UUID {
	t.Helper()
	ctx := context.Background()
	u, err := q.CreateUser(ctx, sqlc.CreateUserParams{
		IsActive: true,
		Metadata: []byte("{}"),
	})
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}
	return u.ID
}

// createTestBot inserts a minimal bot for FK references.
func createTestBot(t *testing.T, q *sqlc.Queries, ownerID pgtype.UUID) pgtype.UUID {
	t.Helper()
	ctx := context.Background()
	b, err := q.CreateBot(ctx, sqlc.CreateBotParams{
		OwnerUserID: ownerID,
		Type:        "personal",
		IsActive:    true,
		Metadata:    []byte("{}"),
		Status:      "ready",
	})
	if err != nil {
		t.Fatalf("create test bot: %v", err)
	}
	return b.ID
}

func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func TestDepartmentCRUD(t *testing.T) {
	q, _, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Create
	dept, err := q.CreateDepartment(ctx, sqlc.CreateDepartmentParams{
		Name:        "Engineering-" + t.Name(),
		Description: "Engineering dept",
		Metadata:    []byte("{}"),
	})
	if err != nil {
		t.Fatalf("CreateDepartment: %v", err)
	}
	if dept.Name != "Engineering-"+t.Name() {
		t.Fatalf("name = %q, want %q", dept.Name, "Engineering-"+t.Name())
	}

	// Get by ID
	got, err := q.GetDepartmentByID(ctx, dept.ID)
	if err != nil {
		t.Fatalf("GetDepartmentByID: %v", err)
	}
	if got.ID != dept.ID {
		t.Fatalf("ID mismatch")
	}

	// Get by Name
	got2, err := q.GetDepartmentByName(ctx, dept.Name)
	if err != nil {
		t.Fatalf("GetDepartmentByName: %v", err)
	}
	if got2.ID != dept.ID {
		t.Fatalf("name lookup ID mismatch")
	}

	// Update
	updated, err := q.UpdateDepartment(ctx, sqlc.UpdateDepartmentParams{
		ID:          dept.ID,
		Name:        "Eng-Updated-" + t.Name(),
		Description: "Updated",
		Metadata:    []byte(`{"key":"value"}`),
	})
	if err != nil {
		t.Fatalf("UpdateDepartment: %v", err)
	}
	if updated.Description != "Updated" {
		t.Fatalf("description = %q, want %q", updated.Description, "Updated")
	}

	// List
	depts, err := q.ListDepartments(ctx)
	if err != nil {
		t.Fatalf("ListDepartments: %v", err)
	}
	found := false
	for _, d := range depts {
		if d.ID == dept.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("department not found in list")
	}

	// Delete
	err = q.DeleteDepartment(ctx, dept.ID)
	if err != nil {
		t.Fatalf("DeleteDepartment: %v", err)
	}
	_, err = q.GetDepartmentByID(ctx, dept.ID)
	if err == nil {
		t.Fatal("expected error after delete, got nil")
	}
}

func TestDepartmentHierarchy(t *testing.T) {
	q, _, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	parent, err := q.CreateDepartment(ctx, sqlc.CreateDepartmentParams{
		Name:        "Parent-" + t.Name(),
		Description: "parent",
		Metadata:    []byte("{}"),
	})
	if err != nil {
		t.Fatal(err)
	}

	child, err := q.CreateDepartment(ctx, sqlc.CreateDepartmentParams{
		Name:        "Child-" + t.Name(),
		Description: "child",
		ParentID:    parent.ID,
		Metadata:    []byte("{}"),
	})
	if err != nil {
		t.Fatal(err)
	}

	children, err := q.ListChildDepartments(ctx, parent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(children) != 1 || children[0].ID != child.ID {
		t.Fatalf("expected 1 child, got %d", len(children))
	}

	// Cleanup
	_ = q.DeleteDepartment(ctx, child.ID)
	_ = q.DeleteDepartment(ctx, parent.ID)
}

func TestDepartmentMembers(t *testing.T) {
	q, _, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	userID := createTestUser(t, q)
	dept, err := q.CreateDepartment(ctx, sqlc.CreateDepartmentParams{
		Name:     "DeptMembers-" + t.Name(),
		Metadata: []byte("{}"),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Add member
	err = q.AddDepartmentMember(ctx, sqlc.AddDepartmentMemberParams{
		DepartmentID: dept.ID,
		UserID:       userID,
		Role:         "admin",
	})
	if err != nil {
		t.Fatalf("AddDepartmentMember: %v", err)
	}

	// Check membership
	isMember, err := q.CheckUserDepartmentMembership(ctx, sqlc.CheckUserDepartmentMembershipParams{
		DepartmentID: dept.ID,
		UserID:       userID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !isMember {
		t.Fatal("expected user to be member")
	}

	// List members
	members, err := q.ListDepartmentMembers(ctx, dept.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(members))
	}

	// List user departments
	userDepts, err := q.ListUserDepartments(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(userDepts) != 1 || userDepts[0].ID != dept.ID {
		t.Fatalf("expected 1 dept for user, got %d", len(userDepts))
	}

	// Remove member
	err = q.RemoveDepartmentMember(ctx, sqlc.RemoveDepartmentMemberParams{
		DepartmentID: dept.ID,
		UserID:       userID,
	})
	if err != nil {
		t.Fatal(err)
	}
	isMember, err = q.CheckUserDepartmentMembership(ctx, sqlc.CheckUserDepartmentMembershipParams{
		DepartmentID: dept.ID,
		UserID:       userID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if isMember {
		t.Fatal("expected user NOT to be member after removal")
	}

	// Cleanup
	_ = q.DeleteDepartment(ctx, dept.ID)
}

func TestBotDepartmentAccess(t *testing.T) {
	q, _, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	userID := createTestUser(t, q)
	botID := createTestBot(t, q, userID)
	dept, err := q.CreateDepartment(ctx, sqlc.CreateDepartmentParams{
		Name:     "BotAccess-" + t.Name(),
		Metadata: []byte("{}"),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Grant access
	err = q.SetBotDepartmentAccess(ctx, sqlc.SetBotDepartmentAccessParams{
		BotID:        botID,
		DepartmentID: dept.ID,
	})
	if err != nil {
		t.Fatalf("SetBotDepartmentAccess: %v", err)
	}

	// Check access
	hasAccess, err := q.CheckBotDepartmentAccess(ctx, sqlc.CheckBotDepartmentAccessParams{
		BotID:        botID,
		DepartmentID: dept.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !hasAccess {
		t.Fatal("expected bot to have access")
	}

	// List bot departments
	botDepts, err := q.ListBotDepartments(ctx, botID)
	if err != nil {
		t.Fatal(err)
	}
	if len(botDepts) != 1 {
		t.Fatalf("expected 1 dept, got %d", len(botDepts))
	}

	// List department bots
	deptBots, err := q.ListDepartmentBots(ctx, dept.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(deptBots) != 1 {
		t.Fatalf("expected 1 bot, got %d", len(deptBots))
	}

	// Revoke access
	err = q.RemoveBotDepartmentAccess(ctx, sqlc.RemoveBotDepartmentAccessParams{
		BotID:        botID,
		DepartmentID: dept.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	hasAccess, err = q.CheckBotDepartmentAccess(ctx, sqlc.CheckBotDepartmentAccessParams{
		BotID:        botID,
		DepartmentID: dept.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if hasAccess {
		t.Fatal("expected bot NOT to have access after revoke")
	}

	// Cleanup
	_ = q.DeleteDepartment(ctx, dept.ID)
}

func TestBotPermissions(t *testing.T) {
	q, _, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	userID := createTestUser(t, q)
	botID := createTestBot(t, q, userID)

	// Upsert
	perm, err := q.UpsertBotPermission(ctx, sqlc.UpsertBotPermissionParams{
		BotID:    botID,
		Role:     "member",
		Resource: "chat",
		Action:   "execute",
		Allowed:  true,
	})
	if err != nil {
		t.Fatalf("UpsertBotPermission: %v", err)
	}
	if !perm.Allowed {
		t.Fatal("expected allowed=true")
	}

	// Update via upsert
	perm2, err := q.UpsertBotPermission(ctx, sqlc.UpsertBotPermissionParams{
		BotID:    botID,
		Role:     "member",
		Resource: "chat",
		Action:   "execute",
		Allowed:  false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if perm2.Allowed {
		t.Fatal("expected allowed=false after upsert update")
	}

	// List
	perms, err := q.ListBotPermissions(ctx, botID)
	if err != nil {
		t.Fatal(err)
	}
	if len(perms) != 1 {
		t.Fatalf("expected 1 permission, got %d", len(perms))
	}

	// Get specific
	got, err := q.GetBotPermission(ctx, sqlc.GetBotPermissionParams{
		BotID:    botID,
		Role:     "member",
		Resource: "chat",
		Action:   "execute",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Allowed {
		t.Fatal("expected allowed=false from get")
	}

	// Delete all by bot
	err = q.DeleteBotPermissionsByBot(ctx, botID)
	if err != nil {
		t.Fatal(err)
	}
	perms, err = q.ListBotPermissions(ctx, botID)
	if err != nil {
		t.Fatal(err)
	}
	if len(perms) != 0 {
		t.Fatalf("expected 0 permissions after delete, got %d", len(perms))
	}
}

func TestAuditLogs(t *testing.T) {
	q, _, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	userID := createTestUser(t, q)

	// Insert
	entry, err := q.InsertAuditLog(ctx, sqlc.InsertAuditLogParams{
		UserID:       userID,
		Action:       "bot.create",
		ResourceType: "bot",
		Detail:       []byte(`{"name":"test"}`),
	})
	if err != nil {
		t.Fatalf("InsertAuditLog: %v", err)
	}
	if entry.Action != "bot.create" {
		t.Fatalf("action = %q", entry.Action)
	}

	// List with filter
	logs, err := q.ListAuditLogs(ctx, sqlc.ListAuditLogsParams{
		UserID:   userID,
		PageSize: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) < 1 {
		t.Fatal("expected at least 1 log")
	}

	// Count
	count, err := q.CountAuditLogs(ctx, sqlc.CountAuditLogsParams{
		UserID: userID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if count < 1 {
		t.Fatalf("expected count >= 1, got %d", count)
	}
}

func TestBudgets(t *testing.T) {
	q, _, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	userID := createTestUser(t, q)
	scopeID := uuidToString(userID)

	budget, err := q.CreateBudget(ctx, sqlc.CreateBudgetParams{
		ScopeType:      "user",
		ScopeID:        scopeID,
		Period:         "monthly",
		LimitAmount:    numericFromString("100.00"),
		AlertThreshold: numericFromString("0.80"),
		ActionOnExceed: "warn",
		IsEnabled:      true,
	})
	if err != nil {
		t.Fatalf("CreateBudget: %v", err)
	}
	if budget.ScopeType != "user" {
		t.Fatalf("scope_type = %q", budget.ScopeType)
	}

	// Get
	got, err := q.GetBudget(ctx, sqlc.GetBudgetParams{
		ScopeType: "user",
		ScopeID:   scopeID,
		Period:    "monthly",
	})
	if err != nil {
		t.Fatalf("GetBudget: %v", err)
	}
	if got.ID != budget.ID {
		t.Fatal("ID mismatch")
	}

	// Delete
	err = q.DeleteBudget(ctx, budget.ID)
	if err != nil {
		t.Fatal(err)
	}
}
