package timesheet

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
	"strconv"

	"finos.com/api/internal/auth"
	"finos.com/api/internal/helper"
)

const maximumCreateBodyBytes = 64 << 10

type TimesheetService interface {
	Create(ctx context.Context, userID int64, input CreateInput) (Response, error)
	GetAll(ctx context.Context, userID int64, userRole string) ([]Response, error)
	GetOptions(ctx context.Context, userID int64, userRole string) ([]Option, error)
	ApproveTimesheet(ctx context.Context, timesheetID int64, userRole string) (Response, error)
}

type Handler struct {
	service TimesheetService
}

type listResponse struct {
	Timesheets []Response `json:"timesheets"`
}

type optionsResponse struct {
	Timesheets []Option `json:"timesheets"`
}

func NewHandler(service TimesheetService) *Handler {
	return &Handler{service: service}
}

func (handler *Handler) Create(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required")
		return
	}

	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(
			w,
			http.StatusUnsupportedMediaType,
			"unsupported_media_type",
			"Content-Type must be application/json",
		)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maximumCreateBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var input CreateInput
	if err := decoder.Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body contains invalid JSON")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must contain one JSON object")
		return
	}

	createdTimesheet, err := handler.service.Create(r.Context(), principal.UserID, input)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidDate), errors.Is(err, ErrInvalidHours):
			writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		case errors.Is(err, ErrInvalidUserID):
			writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required")
		default:
			log.Printf("create timesheet: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		}
		return
	}

	helper.WriteJSON(w, http.StatusCreated, createdTimesheet)
}

func (handler *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required")
		return
	}

	timesheets, err := handler.service.GetAll(r.Context(), principal.UserID, principal.Role)
	if err != nil {
		if errors.Is(err, ErrInvalidUserID) {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required")
			return
		}

		log.Printf("get user timesheets: %v", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		return
	}

	helper.WriteJSON(w, http.StatusOK, listResponse{Timesheets: timesheets})
}

func (handler *Handler) GetOptions(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required")
		return
	}

	options, err := handler.service.GetOptions(r.Context(), principal.UserID, principal.Role)
	if err != nil {
		if errors.Is(err, ErrInvalidUserID) {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required")
			return
		}

		log.Printf("get timesheet options: %v", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	helper.WriteJSON(w, http.StatusOK, optionsResponse{Timesheets: options})
}

func (handler *Handler) ApproveTimesheet(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required")
		return
	}
	timesheetID, err := strconv.ParseInt(r.PathValue("timesheetID"), 10, 64)
	if err != nil || timesheetID <= 0 {
		writeError(w, http.StatusBadRequest, "validation_error", "Timesheet ID must be a positive integer")
		return
	}

	timesheet, err := handler.service.ApproveTimesheet(r.Context(), timesheetID, principal.Role)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotManager):
			writeError(w, http.StatusForbidden, "forbidden", "Only managers can approve timesheets")
		case errors.Is(err, ErrTimesheetNotFound):
			writeError(w, http.StatusNotFound, "not_found", "Timesheet not found")
		default:
			log.Printf("approve timesheet: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		}
		return
	}

	helper.WriteJSON(w, http.StatusOK, timesheet)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	helper.WriteJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
