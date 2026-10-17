package server

import (
	"context"
	"net/http"
	"time"

	"finos.com/api/internal/auth"
	"finos.com/api/internal/helper"
	"finos.com/api/internal/reimbursement"
	"finos.com/api/internal/timesheet"
	"finos.com/api/internal/user"
	"github.com/jackc/pgx/v5/pgxpool"
)

func New(
	database *pgxpool.Pool,
	tokenManager *auth.TokenManager,
	userHandler *user.Handler,
	timesheetHandler *timesheet.Handler,
	reimbursementHandler *reimbursement.Handler,
) http.Handler {
	mux := http.NewServeMux()
	authenticate := auth.Authenticate(tokenManager)

	mux.HandleFunc("GET /docs", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/", http.StatusMovedPermanently)
	})
	mux.HandleFunc("GET /docs/", swaggerUIHandler)
	mux.HandleFunc("GET /openapi.yaml", openAPIHandler)
	mux.HandleFunc("GET /health", healthHandler(database))
	mux.HandleFunc("POST /api/v1/auth/register", userHandler.Register)
	mux.HandleFunc("POST /api/v1/auth/login", userHandler.Login)
	mux.Handle(
		"GET /api/v1/users",
		authenticate(http.HandlerFunc(userHandler.List)),
	)
	mux.Handle(
		"PATCH /api/v1/users/{userID}/promote",
		authenticate(http.HandlerFunc(userHandler.PromoteToManager)),
	)
	mux.Handle(
		"POST /api/v1/timesheets",
		authenticate(http.HandlerFunc(timesheetHandler.Create)),
	)
	mux.Handle(
		"GET /api/v1/timesheets",
		authenticate(http.HandlerFunc(timesheetHandler.GetAll)),
	)
	mux.Handle(
		"GET /api/v1/timesheets/options",
		authenticate(http.HandlerFunc(timesheetHandler.GetOptions)),
	)
	mux.Handle(
		"PATCH /api/v1/timesheets/{timesheetID}/approve",
		authenticate(http.HandlerFunc(timesheetHandler.ApproveTimesheet)),
	)
	mux.Handle(
		"POST /api/v1/timesheets/{timesheetID}/reimbursements",
		authenticate(http.HandlerFunc(reimbursementHandler.Create)),
	)
	mux.Handle(
		"GET /api/v1/timesheets/{timesheetID}/reimbursements",
		authenticate(http.HandlerFunc(reimbursementHandler.ListByTimesheet)),
	)
	mux.Handle(
		"GET /api/v1/reimbursements/{reimbursementID}",
		authenticate(http.HandlerFunc(reimbursementHandler.Get)),
	)
	mux.Handle(
		"GET /api/v1/reimbursements/{reimbursementID}/receipt",
		authenticate(http.HandlerFunc(reimbursementHandler.Receipt)),
	)

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
