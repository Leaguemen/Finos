package user

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
	"time"

	"finos.com/api/internal/auth"
	"finos.com/api/internal/helper"
)

const maximumRegistrationBodyBytes = 64 << 10

type RegistrationService interface {
	Register(ctx context.Context, input RegisterInput) (Response, error)
	Login(ctx context.Context, input LoginInput) (Response, error)
}

type Handler struct {
	service      RegistrationService
	tokenManager *auth.TokenManager
}

type RegisterResponse struct {
	AccessToken string   `json:"access_token"`
	TokenType   string   `json:"token_type"`
	ExpiresIn   int64    `json:"expires_in"`
	User        Response `json:"user"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorResponse struct {
	Error errorBody `json:"error"`
}

func NewHandler(
	service RegistrationService,
	tokenManager *auth.TokenManager,
) *Handler {
	return &Handler{
		service:      service,
		tokenManager: tokenManager,
	}
}

func (handler *Handler) Register(w http.ResponseWriter, r *http.Request) {
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

	r.Body = http.MaxBytesReader(w, r.Body, maximumRegistrationBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var input RegisterInput
	if err := decoder.Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body contains invalid JSON")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must contain one JSON object")
		return
	}

	registeredUser, err := handler.service.Register(r.Context(), input)
	if err != nil {
		handler.writeRegistrationError(w, err)
		return
	}

	accessToken, expiresAt, err := handler.tokenManager.Generate(
		registeredUser.ID,
		string(registeredUser.Role),
	)
	if err != nil {
		log.Printf("generate registration JWT: %v", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	helper.WriteJSON(w, http.StatusCreated, RegisterResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(time.Until(expiresAt).Seconds()),
		User:        registeredUser,
	})
}

func (handler *Handler) writeRegistrationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNameRequired),
		errors.Is(err, ErrNameTooLong),
		errors.Is(err, ErrInvalidEmail),
		errors.Is(err, ErrPasswordTooShort),
		errors.Is(err, ErrPasswordTooLong):
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
	case errors.Is(err, ErrEmailTaken):
		writeError(w, http.StatusConflict, "email_taken", "Email is already registered")
	default:
		log.Printf("register user: %v", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}

func (handler *Handler) Login(w http.ResponseWriter, r *http.Request) {
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

	r.Body = http.MaxBytesReader(w, r.Body, maximumRegistrationBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var input LoginInput
	if err := decoder.Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body contains invalid JSON")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must contain one JSON object")
		return
	}
	loggedInUser, err := handler.service.Login(r.Context(), input)
	if err != nil {
		handler.writeLoginError(w, err)
	}
	accessToken, expiresAt, err := handler.tokenManager.Generate(
		loggedInUser.ID,
		string(loggedInUser.Role),
	)
	if err != nil {
		log.Printf("generate registration JWT: %v", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	helper.WriteJSON(w, http.StatusCreated, RegisterResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(time.Until(expiresAt).Seconds()),
		User:        loggedInUser,
	})
}

func (handler *Handler) writeLoginError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFoundUser),
		errors.Is(err, ErrInvalidCredentials):
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
	default:
		log.Printf("login user: %v", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	helper.WriteJSON(w, status, errorResponse{
		Error: errorBody{
			Code:    code,
			Message: message,
		},
	})
}
