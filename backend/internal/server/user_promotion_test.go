package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"finos.com/api/internal/auth"
	"finos.com/api/internal/user"
)

type promotionRepository struct {
	user.Repository
	called bool
	userID int64
	err    error
}

func (repository *promotionRepository) PromoteToManager(_ context.Context, id int64) (user.User, error) {
	repository.called = true
	repository.userID = id
	return user.User{
		ID: id, Name: "Employee", Email: "employee@example.com",
		Role: user.RoleManager, PasswordHash: "private-password-hash",
	}, repository.err
}

func TestPromoteUserRoute(t *testing.T) {
	tokenManager, err := auth.NewTokenManager("01234567890123456789012345678901", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		id         string
		role       string
		body       string
		repoErr    error
		wantStatus int
		wantCall   bool
	}{
		{name: "manager promotes", id: "7", role: "manager", wantStatus: http.StatusOK, wantCall: true},
		{name: "unauthenticated", id: "7", wantStatus: http.StatusUnauthorized},
		{name: "employee cannot promote self", id: "99", role: "employee", wantStatus: http.StatusForbidden},
		{name: "client role is ignored", id: "7", role: "employee", body: `{"role":"manager"}`, wantStatus: http.StatusForbidden},
		{name: "invalid ID", id: "abc", role: "manager", wantStatus: http.StatusBadRequest},
		{name: "zero ID", id: "0", role: "manager", wantStatus: http.StatusBadRequest},
		{name: "negative ID", id: "-1", role: "manager", wantStatus: http.StatusBadRequest},
		{name: "overflow ID", id: "9223372036854775808", role: "manager", wantStatus: http.StatusBadRequest},
		{name: "missing user", id: "7", role: "manager", repoErr: fmt.Errorf("repository: %w", user.ErrNotFound), wantStatus: http.StatusNotFound, wantCall: true},
		{name: "database error", id: "7", role: "manager", repoErr: errors.New("private database details"), wantStatus: http.StatusInternalServerError, wantCall: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &promotionRepository{err: tt.repoErr}
			handler := user.NewHandler(user.NewService(repository), tokenManager)
			router := New(nil, tokenManager, handler, nil)
			request := httptest.NewRequest(http.MethodPatch, "/api/v1/users/"+tt.id+"/promote", strings.NewReader(tt.body))
			if tt.role != "" {
				token, _, err := tokenManager.Generate(99, tt.role)
				if err != nil {
					t.Fatal(err)
				}
				request.Header.Set("Authorization", "Bearer "+token)
			}
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", recorder.Code, tt.wantStatus, recorder.Body.String())
			}
			if repository.called != tt.wantCall {
				t.Fatalf("repository called = %t, want %t", repository.called, tt.wantCall)
			}
			if tt.wantCall && repository.userID != 7 {
				t.Fatalf("repository ID = %d, want 7", repository.userID)
			}
			if tt.wantStatus == http.StatusOK {
				var response user.Response
				if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				if response.ID != 7 || response.Role != user.RoleManager {
					t.Fatalf("unexpected response: %+v", response)
				}
			}
			if strings.Contains(recorder.Body.String(), "private") || strings.Contains(recorder.Body.String(), "password") {
				t.Fatal("response exposed internal details or password hash")
			}
		})
	}
}
