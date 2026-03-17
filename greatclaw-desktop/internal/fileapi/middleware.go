package fileapi

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// withAuth wraps a handler with Bearer token authentication.
func (s *Server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "missing authorization header")
			return
		}

		token := strings.TrimPrefix(auth, "Bearer ")
		if subtle.ConstantTimeCompare([]byte(token), []byte(s.apiToken)) != 1 {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		next(w, r)
	}
}
