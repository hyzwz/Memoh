package auth

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const (
	claimSubject = "sub"
	claimUserID  = "user_id"
)

// JWTMiddleware returns a JWT auth middleware configured for HS256 tokens.
func JWTMiddleware(secret string, skipper middleware.Skipper) echo.MiddlewareFunc {
	return echojwt.WithConfig(echojwt.Config{
		SigningKey:    []byte(secret),
		SigningMethod: "HS256",
		TokenLookup:   "header:Authorization:Bearer ",
		Skipper:       skipper,
		NewClaimsFunc: func(c echo.Context) jwt.Claims {
			return jwt.MapClaims{}
		},
	})
}

// UserIDFromContext extracts the user id from JWT claims.
func UserIDFromContext(c echo.Context) (string, error) {
	token, ok := c.Get("user").(*jwt.Token)
	if !ok || token == nil || !token.Valid {
		return "", echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", echo.NewHTTPError(http.StatusUnauthorized, "invalid token claims")
	}
	if userID := claimString(claims, claimUserID); userID != "" {
		return userID, nil
	}
	if userID := claimString(claims, claimSubject); userID != "" {
		return userID, nil
	}
	return "", echo.NewHTTPError(http.StatusUnauthorized, "user id missing")
}

// GenerateToken creates a signed JWT for the user.
func GenerateToken(userID, secret string, expiresIn time.Duration) (string, time.Time, error) {
	if strings.TrimSpace(userID) == "" {
		return "", time.Time{}, fmt.Errorf("user id is required")
	}
	if strings.TrimSpace(secret) == "" {
		return "", time.Time{}, fmt.Errorf("jwt secret is required")
	}
	if expiresIn <= 0 {
		return "", time.Time{}, fmt.Errorf("jwt expires in must be positive")
	}

	now := time.Now().UTC()
	expiresAt := now.Add(expiresIn)
	claims := jwt.MapClaims{
		claimSubject: userID,
		claimUserID:  userID,
		"iat":        now.Unix(),
		"exp":        expiresAt.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expiresAt, nil
}

func claimString(claims jwt.MapClaims, key string) string {
	raw, ok := claims[key]
	if !ok || raw == nil {
		return ""
	}
	switch v := raw.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprint(raw)
	}
}
