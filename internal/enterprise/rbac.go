package enterprise

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/memohai/memoh/internal/auth"
	"github.com/memohai/memoh/internal/db"
	"github.com/memohai/memoh/internal/db/sqlc"
)

// roleLevel returns a numeric privilege level for a user_role enum value.
// Higher numbers = more privilege.
// validRoles enumerates all known role strings for validation.
var validRoles = map[string]int{
	"member":      0,
	"dept_admin":  1,
	"admin":       2,
	"org_admin":   3,
	"super_admin": 4,
}

func roleLevel(role string) int {
	if level, ok := validRoles[role]; ok {
		return level
	}
	return -1
}

// hasMinRole checks if userRole meets or exceeds the minimum required role.
// Returns false if either role string is unknown (fail closed).
func hasMinRole(userRole, minRequired string) bool {
	userLvl := roleLevel(userRole)
	minLvl := roleLevel(minRequired)
	if userLvl < 0 || minLvl < 0 {
		return false
	}
	return userLvl >= minLvl
}

// RequireRole returns an Echo middleware that enforces a minimum role level.
// It extracts the user ID from JWT, looks up the user's role, and rejects
// requests from users below the required role.
func RequireRole(queries *sqlc.Queries, logger *slog.Logger, minRole string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userID, err := auth.UserIDFromContext(c)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
			}

			pgUserID, err := db.ParseUUID(userID)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid user id")
			}

			user, err := queries.GetUserByID(c.Request().Context(), pgUserID)
			if err != nil {
				logger.Error("rbac: failed to get user", slog.String("user_id", userID), slog.String("error", err.Error()))
				return echo.NewHTTPError(http.StatusForbidden, "access denied")
			}

			if !hasMinRole(user.Role, minRole) {
				return echo.NewHTTPError(http.StatusForbidden, "insufficient permissions")
			}

			// Store role in context for handlers to use.
			c.Set("user_role", user.Role)
			return next(c)
		}
	}
}
