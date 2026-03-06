package handlers

import "context"

// AuditLoggerInterface abstracts audit log writing for handlers.
type AuditLoggerInterface interface {
	Log(ctx context.Context, userID, botID, action, resourceType, resourceID, ipAddress, userAgent string, detail any)
}

// noopAuditLogger is a no-op implementation used when no audit logger is configured.
type noopAuditLogger struct{}

func (noopAuditLogger) Log(context.Context, string, string, string, string, string, string, string, any) {
}
