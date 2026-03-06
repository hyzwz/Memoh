package handlers

import (
	"context"

	"github.com/labstack/echo/v4"

	"github.com/memohai/memoh/internal/auth"
)

// AuditLoggerInterface abstracts audit log writing for handlers.
type AuditLoggerInterface interface {
	Log(ctx context.Context, userID, botID, action, resourceType, resourceID, ipAddress, userAgent string, detail any)
}

// noopAuditLogger is a no-op implementation used when no audit logger is configured.
type noopAuditLogger struct{}

func (noopAuditLogger) Log(context.Context, string, string, string, string, string, string, string, any) {
}

// auditUserID extracts the authenticated user ID from the echo context.
// Returns empty string if not authenticated (best-effort).
func auditUserID(c echo.Context) string {
	userID, _ := auth.UserIDFromContext(c)
	return userID
}
