package timesheet

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"finos.com/api/internal/auth"
)

func TestDeleteTimesheetHTTP(t *testing.T) {
	tokenManager, err := auth.NewTokenManager(handlerTestSecret, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, id, role string
		userID         int64
		want           int
		called         bool
	}{
		{"owner", "7", "employee", 42, 204, true},
		{"manager owner", "7", "manager", 42, 204, true},
		{"manager other", "7", "manager", 99, 403, false},
		{"invalid ID", "bad", "employee", 42, 400, false},
		{"anonymous", "7", "", 0, 401, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repository := &stubRepository{result: Timesheet{ID: 7, UserID: 42}}
			handler := NewHandler(NewService(repository, nil))
			mux := http.NewServeMux()
			mux.Handle("DELETE /api/v1/timesheets/{timesheetID}", auth.Authenticate(tokenManager)(http.HandlerFunc(handler.Delete)))
			request := httptest.NewRequest(http.MethodDelete, "/api/v1/timesheets/"+tc.id, nil)
			if tc.role != "" {
				token, _, err := tokenManager.Generate(tc.userID, tc.role)
				if err != nil {
					t.Fatal(err)
				}
				request.Header.Set("Authorization", "Bearer "+token)
			}
			recorder := httptest.NewRecorder()
			mux.ServeHTTP(recorder, request)
			if recorder.Code != tc.want || repository.deleteCalled != tc.called {
				t.Fatalf("status = %d, deleted = %t, body = %s", recorder.Code, repository.deleteCalled, recorder.Body.String())
			}
			if tc.want == 204 && recorder.Body.Len() != 0 {
				t.Fatalf("204 response has body: %s", recorder.Body.String())
			}
		})
	}
}
