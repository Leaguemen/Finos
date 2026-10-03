package timesheet

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"

	"finos.com/api/internal/auth"
	"finos.com/api/internal/helper"
)

const maximumCreateBodyBytes = 64 << 10

type TimesheetService interface {
	Create(ctx context.Context, userID int64, input CreateInput) (Response, error)
	GetAll(ctx context.Context, userID int64) ([]Response, error)
}

type Handler struct {
	service TimesheetService
}

type listResponse struct {
	Timesheets []Response `json:"timesheets"`
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

	timesheets, err := handler.service.GetAll(r.Context(), principal.UserID)
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

func writeError(w http.ResponseWriter, status int, code, message string) {
	helper.WriteJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
