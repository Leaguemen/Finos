package timesheet

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"finos.com/api/internal/auth"
)

const handlerTestSecret = "01234567890123456789012345678901"

type stubCreator struct {
	userID     int64
	input      CreateInput
	result     Response
	listResult []Response
	called     bool
	listCalled bool
}

func (creator *stubCreator) ApproveTimesheet(_ context.Context, _ int64, _ string) (Response, error) {
	panic("unexpected ApproveTimesheet call")
}

func (creator *stubCreator) Create(
	_ context.Context,
	userID int64,
	input CreateInput,
) (Response, error) {
	creator.called = true
	creator.userID = userID
	creator.input = input
	return creator.result, nil
}

func (creator *stubCreator) GetAll(
	_ context.Context,
	userID int64,
	_ string,
) ([]Response, error) {
	creator.listCalled = true
	creator.userID = userID
	return creator.listResult, nil
}

func (creator *stubCreator) GetOptions(
	_ context.Context,
	userID int64,
	_ string,
) ([]Option, error) {
	creator.listCalled = true
	creator.userID = userID
	options := make([]Option, 0, len(creator.listResult))
	for _, timesheet := range creator.listResult {
		options = append(options, Option{ID: timesheet.ID, Description: timesheet.Description})
	}
	return options, nil
}

func TestCreateGetsUserIDFromJWT(t *testing.T) {
	manager, err := auth.NewTokenManager(handlerTestSecret, time.Hour)
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}
	token, _, err := manager.Generate(42, "employee")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	creator := &stubCreator{result: Response{ID: 7, UserID: 42}}
	handler := NewHandler(creator)
	protectedHandler := auth.Authenticate(manager)(http.HandlerFunc(handler.Create))

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/timesheets",
		strings.NewReader(`{
			"work_date":"2026-10-03",
			"hours_worked":8,
			"description":"Worked on API"
		}`),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	protectedHandler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
	if creator.userID != 42 {
		t.Fatalf("service user ID = %d, want JWT user ID 42", creator.userID)
	}
	if creator.input.WorkDate != "2026-10-03" || creator.input.HoursWorked != 8 {
		t.Fatalf("service input = %+v", creator.input)
	}
}

func TestCreateRejectsClientSuppliedUserID(t *testing.T) {
	manager, err := auth.NewTokenManager(handlerTestSecret, time.Hour)
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}
	token, _, err := manager.Generate(42, "employee")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	creator := &stubCreator{}
	handler := NewHandler(creator)
	protectedHandler := auth.Authenticate(manager)(http.HandlerFunc(handler.Create))

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/timesheets",
		strings.NewReader(`{
			"user_id":999,
			"work_date":"2026-10-03",
			"hours_worked":8
		}`),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	protectedHandler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if creator.called {
		t.Fatal("service was called with a client-supplied user_id")
	}
}

func TestGetAllGetsUserIDFromJWT(t *testing.T) {
	manager, err := auth.NewTokenManager(handlerTestSecret, time.Hour)
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}
	token, _, err := manager.Generate(42, "employee")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	creator := &stubCreator{
		listResult: []Response{{ID: 7, UserID: 42, WorkDate: "2026-10-03"}},
	}
	handler := NewHandler(creator)
	protectedHandler := auth.Authenticate(manager)(http.HandlerFunc(handler.GetAll))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/timesheets", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	protectedHandler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if !creator.listCalled || creator.userID != 42 {
		t.Fatalf("service user ID = %d, want JWT user ID 42", creator.userID)
	}
	if !strings.Contains(recorder.Body.String(), `"timesheets"`) {
		t.Fatalf("body = %s, want timesheets field", recorder.Body.String())
	}
}

func TestGetOptionsGetsUserIDFromJWT(t *testing.T) {
	manager, err := auth.NewTokenManager(handlerTestSecret, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := manager.Generate(42, "manager")
	if err != nil {
		t.Fatal(err)
	}
	description := "October work"
	creator := &stubCreator{
		listResult: []Response{{ID: 7, Description: &description}},
	}
	handler := NewHandler(creator)
	protectedHandler := auth.Authenticate(manager)(http.HandlerFunc(handler.GetOptions))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/timesheets/options", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	protectedHandler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if !creator.listCalled || creator.userID != 42 {
		t.Fatalf("service user ID = %d, want JWT user ID 42", creator.userID)
	}
	if recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", recorder.Header().Get("Cache-Control"))
	}
	if !strings.Contains(recorder.Body.String(), `"id":7`) ||
		!strings.Contains(recorder.Body.String(), `"description":"October work"`) {
		t.Fatalf("body = %s", recorder.Body.String())
	}
}
