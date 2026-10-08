package timesheet

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"finos.com/api/internal/auth"
	"github.com/jackc/pgx/v5"
)

func TestApproveTimesheet(t *testing.T) {
	tokenManager, err := auth.NewTokenManager(handlerTestSecret, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name         string
		id           string
		role         string
		repoErr      error
		wantStatus   int
		wantCode     string
		wantCall     bool
		resultStatus Status
	}{
		{name: "manager approves", id: "7", role: "manager", wantStatus: http.StatusOK, wantCall: true, resultStatus: StatusApproved},
		{name: "manager reverts approval", id: "7", role: "manager", wantStatus: http.StatusOK, wantCall: true, resultStatus: StatusDraft},
		{name: "missing authentication", id: "7", wantStatus: http.StatusUnauthorized, wantCode: "unauthorized"},
		{name: "employee denied", id: "7", role: "employee", wantStatus: http.StatusForbidden, wantCode: "forbidden"},
		{name: "invalid ID", id: "abc", role: "manager", wantStatus: http.StatusBadRequest, wantCode: "validation_error"},
		{name: "zero ID", id: "0", role: "manager", wantStatus: http.StatusBadRequest, wantCode: "validation_error"},
		{name: "negative ID", id: "-1", role: "manager", wantStatus: http.StatusBadRequest, wantCode: "validation_error"},
		{name: "overflow ID", id: "9223372036854775808", role: "manager", wantStatus: http.StatusBadRequest, wantCode: "validation_error"},
		{name: "not found", id: "7", role: "manager", repoErr: fmt.Errorf("approve timesheet: %w", pgx.ErrNoRows), wantStatus: http.StatusNotFound, wantCode: "not_found", wantCall: true},
		{name: "database error", id: "7", role: "manager", repoErr: errors.New("private database details"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error", wantCall: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &stubRepository{
				result: Timesheet{ID: 7, UserID: 42, Status: tt.resultStatus},
				err:    tt.repoErr,
			}
			handler := NewHandler(NewService(repository))
			mux := http.NewServeMux()
			mux.Handle("PATCH /api/v1/timesheets/{timesheetID}/approve",
				auth.Authenticate(tokenManager)(http.HandlerFunc(handler.ApproveTimesheet)))

			request := httptest.NewRequest(http.MethodPatch, "/api/v1/timesheets/"+tt.id+"/approve", nil)
			if tt.role != "" {
				token, _, err := tokenManager.Generate(99, tt.role)
				if err != nil {
					t.Fatal(err)
				}
				request.Header.Set("Authorization", "Bearer "+token)
			}
			recorder := httptest.NewRecorder()
			mux.ServeHTTP(recorder, request)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", recorder.Code, tt.wantStatus, recorder.Body.String())
			}
			if repository.approveCalled != tt.wantCall {
				t.Fatalf("repository called = %t, want %t", repository.approveCalled, tt.wantCall)
			}
			if tt.wantCall && repository.approvedID != 7 {
				t.Fatalf("repository ID = %d, want 7", repository.approvedID)
			}
			if tt.wantStatus == http.StatusOK {
				var response Response
				if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				if response.ID != 7 || response.Status != tt.resultStatus {
					t.Fatalf("unexpected response: %+v", response)
				}
			} else {
				var response struct {
					Error struct {
						Code string `json:"code"`
					} `json:"error"`
				}
				if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				if response.Error.Code != tt.wantCode {
					t.Fatalf("error code = %q, want %q", response.Error.Code, tt.wantCode)
				}
			}
			if strings.Contains(recorder.Body.String(), "private database details") {
				t.Fatal("response exposed internal error details")
			}
		})
	}
}
