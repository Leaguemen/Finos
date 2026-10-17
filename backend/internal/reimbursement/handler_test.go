package reimbursement

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"finos.com/api/internal/auth"
)

const handlerTestSecret = "01234567890123456789012345678901"

type stubService struct {
	userID      int64
	timesheetID int64
	input       CreateInput
	result      Response
	url         string
	called      bool
	err         error
}

func (service *stubService) Create(
	_ context.Context,
	userID int64,
	timesheetID int64,
	input CreateInput,
) (Response, error) {
	service.called = true
	service.userID = userID
	service.timesheetID = timesheetID
	service.input = input
	return service.result, service.err
}

func (service *stubService) ListByTimesheet(context.Context, int64, string, int64) ([]Response, error) {
	return nil, service.err
}

func (service *stubService) Get(context.Context, int64, string, int64) (Response, error) {
	return service.result, service.err
}

func (service *stubService) ReceiptURL(context.Context, int64, string, int64) (string, error) {
	return service.url, service.err
}

func TestCreateReimbursementUsesJWTIdentityAndAcceptsReceipt(t *testing.T) {
	tokenManager, token := reimbursementTestToken(t, 42, "employee")
	service := &stubService{result: Response{ID: 23, UserID: 42, TimesheetID: 12}}
	handler := NewHandler(service)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writeField(t, writer, "expense_date", "2026-10-17")
	writeField(t, writer, "amount", "125000.00")
	writeField(t, writer, "category", "transportation")
	fileWriter, err := writer.CreateFormFile("receipt", "receipt.png")
	if err != nil {
		t.Fatal(err)
	}
	imageData := image.NewRGBA(image.Rect(0, 0, 2, 2))
	imageData.Set(0, 0, color.White)
	if err := png.Encode(fileWriter, imageData); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/timesheets/12/reimbursements", body)
	request.SetPathValue("timesheetID", "12")
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	auth.Authenticate(tokenManager)(http.HandlerFunc(handler.Create)).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
	if !service.called || service.userID != 42 || service.timesheetID != 12 {
		t.Fatalf("service call: called=%t userID=%d timesheetID=%d", service.called, service.userID, service.timesheetID)
	}
	if service.input.Receipt == nil || service.input.Receipt.ContentType != "image/png" {
		t.Fatalf("receipt input = %+v", service.input.Receipt)
	}
	if recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q", recorder.Header().Get("Cache-Control"))
	}
}

func TestCreateReimbursementRejectsClientIdentityField(t *testing.T) {
	tokenManager, token := reimbursementTestToken(t, 42, "employee")
	service := &stubService{}
	handler := NewHandler(service)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writeField(t, writer, "expense_date", "2026-10-17")
	writeField(t, writer, "amount", "10.00")
	writeField(t, writer, "category", "meal")
	writeField(t, writer, "user_id", "999")
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/timesheets/12/reimbursements", body)
	request.SetPathValue("timesheetID", "12")
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	auth.Authenticate(tokenManager)(http.HandlerFunc(handler.Create)).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if service.called {
		t.Fatal("service was called with client-provided identity")
	}
}

func TestReceiptRedirectRequiresAuthorizationAndDisablesCaching(t *testing.T) {
	tokenManager, token := reimbursementTestToken(t, 42, "employee")
	service := &stubService{url: "https://api.cloudinary.example/signed"}
	handler := NewHandler(service)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/reimbursements/23/receipt", nil)
	request.SetPathValue("reimbursementID", "23")
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	auth.Authenticate(tokenManager)(http.HandlerFunc(handler.Receipt)).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusFound)
	}
	if recorder.Header().Get("Location") != service.url {
		t.Fatalf("Location = %q", recorder.Header().Get("Location"))
	}
	if recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q", recorder.Header().Get("Cache-Control"))
	}
}

func reimbursementTestToken(t *testing.T, userID int64, role string) (*auth.TokenManager, string) {
	t.Helper()
	manager, err := auth.NewTokenManager(handlerTestSecret, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := manager.Generate(userID, role)
	if err != nil {
		t.Fatal(err)
	}
	return manager, token
}

func writeField(t *testing.T, writer *multipart.Writer, name, value string) {
	t.Helper()
	if err := writer.WriteField(name, value); err != nil {
		t.Fatal(err)
	}
}
