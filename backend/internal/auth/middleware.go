package auth

import (
	"net/http"
	"strconv"
	"strings"

	"finos.com/api/internal/helper"
)

// Authenticate verifies a bearer token and makes its identity available to
// downstream handlers through PrincipalFromContext.
func Authenticate(tokenManager *TokenManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r.Header.Get("Authorization"))
			if !ok {
				writeUnauthorized(w)
				return
			}

			claims, err := tokenManager.Parse(token)
			if err != nil {
				writeUnauthorized(w)
				return
			}

			userID, err := strconv.ParseInt(claims.Subject, 10, 64)
			if err != nil || userID <= 0 {
				writeUnauthorized(w)
				return
			}

			principal := Principal{
				UserID: userID,
				Role:   claims.Role,
			}
			next.ServeHTTP(w, r.WithContext(withPrincipal(r.Context(), principal)))
		})
	}
}

func bearerToken(header string) (string, bool) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}

	return parts[1], parts[1] != ""
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	helper.WriteJSON(w, http.StatusUnauthorized, map[string]any{
		"error": map[string]string{
			"code":    "unauthorized",
			"message": "A valid bearer token is required",
		},
	})
}
