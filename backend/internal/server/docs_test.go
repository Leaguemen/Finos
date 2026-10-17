package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAPIDocumentationHandlers(t *testing.T) {
	t.Run("serves embedded specification", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
		recorder := httptest.NewRecorder()

		openAPIHandler(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
		}
		if contentType := recorder.Header().Get("Content-Type"); !strings.Contains(contentType, "application/yaml") {
			t.Fatalf("Content-Type = %q, want application/yaml", contentType)
		}
		for _, expected := range []string{
			"openapi: 3.0.3",
			"/api/v1/auth/register:",
			"/api/v1/users/{userID}/promote:",
			"/api/v1/timesheets/{timesheetID}/approve:",
			"bearerAuth:",
		} {
			if !strings.Contains(recorder.Body.String(), expected) {
				t.Errorf("OpenAPI specification does not contain %q", expected)
			}
		}
	})

	t.Run("serves Swagger UI", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/docs/", nil)
		recorder := httptest.NewRecorder()

		swaggerUIHandler(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
		}
		if contentType := recorder.Header().Get("Content-Type"); !strings.Contains(contentType, "text/html") {
			t.Fatalf("Content-Type = %q, want text/html", contentType)
		}
		if !strings.Contains(recorder.Body.String(), `url: "/openapi.yaml"`) {
			t.Fatal("Swagger UI does not reference the embedded OpenAPI specification")
		}
	})
}
