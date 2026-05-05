package api

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
)

type Middleware func(http.Handler) http.Handler

// RequireToken returns middleware that requires "Authorization: Bearer <expected>".
// An empty expected token rejects all requests (fail-closed) to avoid accidental open access.
func RequireToken(expected string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if expected == "" || !strings.HasPrefix(h, "Bearer ") {
				writeError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid Authorization header")
				return
			}
			got := strings.TrimPrefix(h, "Bearer ")
			if subtle.ConstantTimeCompare([]byte(got), []byte(expected)) != 1 {
				writeError(w, http.StatusUnauthorized, "unauthorized", "invalid token")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type errorBody struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorBody{Error: msg, Code: code})
}
