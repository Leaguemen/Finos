package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const middlewareTestSecret = "01234567890123456789012345678901"

func TestAuthenticateAddsJWTPrincipalToContext(t *testing.T) {
	manager, err := NewTokenManager(middlewareTestSecret, time.Hour)
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}
	token, _, err := manager.Generate(42, "employee")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	var received Principal
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ok bool
		received, ok = PrincipalFromContext(r.Context())
		if !ok {
			t.Error("PrincipalFromContext() did not find a principal")
		}
		w.WriteHeader(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/timesheets", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	Authenticate(manager)(next).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if received.UserID != 42 || received.Role != "employee" {
		t.Fatalf("principal = %+v", received)
	}
}

func TestAuthenticateRejectsMissingAndInvalidTokens(t *testing.T) {
	manager, err := NewTokenManager(middlewareTestSecret, time.Hour)
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}

	for _, authorization := range []string{"", "Bearer invalid-token", "Basic abc"} {
		t.Run(authorization, func(t *testing.T) {
			called := false
			next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				called = true
			})
			request := httptest.NewRequest(http.MethodPost, "/api/v1/timesheets", nil)
			if authorization != "" {
				request.Header.Set("Authorization", authorization)
			}
			recorder := httptest.NewRecorder()

			Authenticate(manager)(next).ServeHTTP(recorder, request)

			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
			}
			if called {
				t.Fatal("next handler was called for an unauthorized request")
			}
		})
	}
}
