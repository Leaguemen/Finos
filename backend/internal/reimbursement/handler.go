package reimbursement

import (
	"context"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"finos.com/api/internal/auth"
	"finos.com/api/internal/helper"
)

const (
	maximumReceiptBytes = 5 << 20
	maximumRequestBytes = 6 << 20
	maximumImagePixels  = 40_000_000
)

var allowedReceiptTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
}

type ReimbursementService interface {
	Create(ctx context.Context, userID, timesheetID int64, input CreateInput) (Response, error)
	ListByTimesheet(ctx context.Context, userID int64, userRole string, timesheetID int64) ([]Response, error)
	Get(ctx context.Context, userID int64, userRole string, reimbursementID int64) (Response, error)
	ReceiptURL(ctx context.Context, userID int64, userRole string, reimbursementID int64) (string, error)
}

type Handler struct {
	service ReimbursementService
}

type listResponse struct {
	Reimbursements []Response `json:"reimbursements"`
}

func NewHandler(service ReimbursementService) *Handler {
	return &Handler{service: service}
}

func (handler *Handler) Create(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required")
		return
	}

	timesheetID, ok := positivePathID(w, r.PathValue("timesheetID"), "Timesheet")
	if !ok {
		return
	}

	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "multipart/form-data" {
		writeError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be multipart/form-data")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maximumRequestBytes)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		var maximumBytesError *http.MaxBytesError
		if errors.As(err, &maximumBytesError) {
			writeError(w, http.StatusRequestEntityTooLarge, "request_too_large", "Request body must not exceed 6 MiB")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_multipart", "Request body contains invalid multipart form data")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	if err := validateMultipartFields(r.MultipartForm); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	expenseDate, err := singleFormValue(r.MultipartForm, "expense_date", true)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	amount, err := singleFormValue(r.MultipartForm, "amount", true)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	category, err := singleFormValue(r.MultipartForm, "category", true)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	descriptionValue, err := singleFormValue(r.MultipartForm, "description", false)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	var description *string
	if descriptionValue != "" {
		description = &descriptionValue
	}

	var receipt *ReceiptUpload
	file, header, err := r.FormFile("receipt")
	switch {
	case err == nil:
		defer file.Close()
		contentType, validationErr := validateReceipt(file, header)
		if validationErr != nil {
			writeError(w, http.StatusBadRequest, "validation_error", validationErr.Error())
			return
		}
		receipt = &ReceiptUpload{Reader: file, ContentType: contentType}
	case errors.Is(err, http.ErrMissingFile):
	case err != nil:
		writeError(w, http.StatusBadRequest, "invalid_multipart", "Receipt could not be read")
		return
	}

	created, err := handler.service.Create(r.Context(), principal.UserID, timesheetID, CreateInput{
		ExpenseDate: expenseDate,
		Amount:      amount,
		Category:    category,
		Description: description,
		Receipt:     receipt,
	})
	if err != nil {
		handler.writeServiceError(w, "create reimbursement", err)
		return
	}

	writeSensitiveJSON(w, http.StatusCreated, created)
}

func (handler *Handler) ListByTimesheet(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required")
		return
	}
	timesheetID, ok := positivePathID(w, r.PathValue("timesheetID"), "Timesheet")
	if !ok {
		return
	}

	reimbursements, err := handler.service.ListByTimesheet(
		r.Context(), principal.UserID, principal.Role, timesheetID,
	)
	if err != nil {
		handler.writeServiceError(w, "list reimbursements", err)
		return
	}
	writeSensitiveJSON(w, http.StatusOK, listResponse{Reimbursements: reimbursements})
}

func (handler *Handler) Get(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required")
		return
	}
	reimbursementID, ok := positivePathID(w, r.PathValue("reimbursementID"), "Reimbursement")
	if !ok {
		return
	}

	reimbursement, err := handler.service.Get(
		r.Context(), principal.UserID, principal.Role, reimbursementID,
	)
	if err != nil {
		handler.writeServiceError(w, "get reimbursement", err)
		return
	}
	writeSensitiveJSON(w, http.StatusOK, reimbursement)
}

func (handler *Handler) Receipt(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required")
		return
	}
	reimbursementID, ok := positivePathID(w, r.PathValue("reimbursementID"), "Reimbursement")
	if !ok {
		return
	}

	url, err := handler.service.ReceiptURL(
		r.Context(), principal.UserID, principal.Role, reimbursementID,
	)
	if err != nil {
		handler.writeServiceError(w, "get reimbursement receipt", err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, url, http.StatusFound)
}

func (handler *Handler) writeServiceError(w http.ResponseWriter, operation string, err error) {
	switch {
	case errors.Is(err, ErrInvalidDate),
		errors.Is(err, ErrInvalidAmount),
		errors.Is(err, ErrCategoryRequired),
		errors.Is(err, ErrCategoryTooLong),
		errors.Is(err, ErrDescriptionTooLong),
		errors.Is(err, ErrInvalidTimesheetID):
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
	case errors.Is(err, ErrTimesheetNotFound), errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "Reimbursement or timesheet not found")
	case errors.Is(err, ErrReceiptNotFound):
		writeError(w, http.StatusNotFound, "not_found", "Receipt not found")
	case errors.Is(err, ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "You cannot access this reimbursement")
	case errors.Is(err, ErrInvalidUserID):
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required")
	default:
		log.Printf("%s: %v", operation, err)
		writeError(w, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}

func validateMultipartFields(form *multipart.Form) error {
	if form == nil {
		return errors.New("multipart form is required")
	}
	allowedValues := map[string]bool{
		"expense_date": true,
		"amount":       true,
		"category":     true,
		"description":  true,
	}
	for field := range form.Value {
		if !allowedValues[field] {
			return errors.New("unknown form field: " + field)
		}
	}
	for field, files := range form.File {
		if field != "receipt" {
			return errors.New("unknown file field: " + field)
		}
		if len(files) > 1 {
			return errors.New("receipt must contain at most one file")
		}
	}
	return nil
}

func singleFormValue(form *multipart.Form, name string, required bool) (string, error) {
	values := form.Value[name]
	if len(values) > 1 {
		return "", errors.New(name + " must be provided once")
	}
	if len(values) == 0 || (required && strings.TrimSpace(values[0]) == "") {
		if required {
			return "", errors.New(name + " is required")
		}
		return "", nil
	}
	return values[0], nil
}

func validateReceipt(file multipart.File, header *multipart.FileHeader) (string, error) {
	if header.Size <= 0 {
		return "", errors.New("receipt must not be empty")
	}
	if header.Size > maximumReceiptBytes {
		return "", errors.New("receipt must not exceed 5 MiB")
	}

	buffer := make([]byte, 512)
	n, err := io.ReadFull(file, buffer)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", errors.New("receipt could not be read")
	}
	contentType := http.DetectContentType(buffer[:n])
	if !allowedReceiptTypes[contentType] {
		return "", errors.New("receipt must be a JPEG or PNG image")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", errors.New("receipt could not be read")
	}
	config, _, err := image.DecodeConfig(file)
	if err != nil || config.Width <= 0 || config.Height <= 0 {
		return "", errors.New("receipt contains an invalid image")
	}
	if int64(config.Width)*int64(config.Height) > maximumImagePixels {
		return "", errors.New("receipt image dimensions are too large")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", errors.New("receipt could not be read")
	}
	return contentType, nil
}

func positivePathID(w http.ResponseWriter, raw, resource string) (int64, bool) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "validation_error", resource+" ID must be a positive integer")
		return 0, false
	}
	return id, true
}

func writeSensitiveJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Cache-Control", "no-store")
	helper.WriteJSON(w, status, value)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	helper.WriteJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
