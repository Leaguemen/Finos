package server

import (
	"context"
	"net/http"
	"time"

	"finos.com/api/internal/helper"
	"finos.com/api/internal/user"
	"github.com/jackc/pgx/v5/pgxpool"
)

func New(database *pgxpool.Pool, userHandler *user.Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", healthHandler(database))
	mux.HandleFunc("POST /api/v1/auth/register", userHandler.Register)

	return requestLogger(mux)
}

func healthHandler(database *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := database.Ping(ctx); err != nil {
			helper.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{
				"status":   "error",
				"database": "unavailable",
			})
			return
		}

		helper.WriteJSON(w, http.StatusOK, map[string]string{
			"status":   "ok",
			"database": "connected",
		})
	}
}
