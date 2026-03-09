package enterprise

import (
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/memohai/memoh/internal/auth"
	authz "github.com/memohai/memoh/internal/rbac"
)

// BotAuthorizer is the subset of RBAC resolver behavior needed by HTTP middleware.
type BotAuthorizer interface {
	Authorize(ctx echo.Context, userID, botID string, resource authz.Resource, action authz.Action) error
}

// BotActionResolver maps the current request to the RBAC action that should be enforced.
type BotActionResolver func(c echo.Context) (authz.Action, error)

type botAuthorizerAdapter struct {
	resolver *authz.Resolver
}

func (a *botAuthorizerAdapter) Authorize(c echo.Context, userID, botID string, resource authz.Resource, action authz.Action) error {
	return a.resolver.Authorize(c.Request().Context(), userID, botID, resource, action)
}

// NewBotAuthorizerAdapter wraps the shared RBAC resolver for Echo middleware.
func NewBotAuthorizerAdapter(resolver *authz.Resolver) BotAuthorizer {
	if resolver == nil {
		return nil
	}
	return &botAuthorizerAdapter{resolver: resolver}
}

// RequireBotPermission enforces bot-scoped RBAC for enterprise routes.
func RequireBotPermission(authorizer BotAuthorizer, resource authz.Resource, resolveAction BotActionResolver) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if authorizer == nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "authorization not configured")
			}

			userID, err := auth.UserIDFromContext(c)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
			}

			botID := strings.TrimSpace(c.Param("bot_id"))
			if botID == "" {
				return echo.NewHTTPError(http.StatusBadRequest, "bot_id is required")
			}

			action, err := resolveAction(c)
			if err != nil {
				return err
			}

			if err := authorizer.Authorize(c, userID, botID, resource, action); err != nil {
				switch {
				case errors.Is(err, authz.ErrAccessDenied):
					return echo.NewHTTPError(http.StatusForbidden, "insufficient permissions")
				case errors.Is(err, authz.ErrUserNotFound):
					return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
				default:
					return echo.NewHTTPError(http.StatusInternalServerError, "authorization failed")
				}
			}

			return next(c)
		}
	}
}

// ResolveMethodAction maps HTTP verbs to a read/write action pair.
func ResolveMethodAction(readAction, writeAction authz.Action) BotActionResolver {
	return func(c echo.Context) (authz.Action, error) {
		switch c.Request().Method {
		case http.MethodGet, http.MethodHead:
			return readAction, nil
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			return writeAction, nil
		default:
			return "", echo.NewHTTPError(http.StatusMethodNotAllowed, "method not allowed")
		}
	}
}

// ResolveHandsAction maps Hands routes to tools read/write/execute permissions.
func ResolveHandsAction(c echo.Context) (authz.Action, error) {
	switch c.Request().Method {
	case http.MethodGet, http.MethodHead:
		return authz.ActionRead, nil
	case http.MethodDelete:
		return authz.ActionWrite, nil
	case http.MethodPost:
		if strings.TrimSpace(c.Param("hand_id")) != "" {
			return authz.ActionExecute, nil
		}
		return authz.ActionWrite, nil
	default:
		return "", echo.NewHTTPError(http.StatusMethodNotAllowed, "method not allowed")
	}
}

// ResolveCockpitAction maps Cockpit routes to settings read/write permissions.
func ResolveCockpitAction(c echo.Context) (authz.Action, error) {
	switch c.Request().Method {
	case http.MethodGet, http.MethodHead:
		return authz.ActionRead, nil
	case http.MethodPost:
		return authz.ActionWrite, nil
	default:
		return "", echo.NewHTTPError(http.StatusMethodNotAllowed, "method not allowed")
	}
}
