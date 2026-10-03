package timesheet

import (
	"context"
	"errors"
	"testing"
	"time"
)

type stubRepository struct {
	params     CreateTimesheetParams
	result     Timesheet
	listResult []Timesheet
	err        error
	called     bool
	listUserID int64
	listCalled bool
}

func (repository *stubRepository) Create(
	_ context.Context,
	params CreateTimesheetParams,
) (Timesheet, error) {
	repository.called = true
	repository.params = params
	return repository.result, repository.err
}

func (repository *stubRepository) FindAllByUserID(
	_ context.Context,
	userID int64,
) ([]Timesheet, error) {
	repository.listCalled = true
	repository.listUserID = userID
	return repository.listResult, repository.err
}

func TestServiceCreateUsesAuthenticatedUserAndDraftStatus(t *testing.T) {
	description := "  Worked on API  "
	createdAt := time.Date(2026, time.October, 3, 4, 0, 0, 0, time.UTC)
	repository := &stubRepository{
		result: Timesheet{
			ID:          7,
			UserID:      42,
			WorkDate:    time.Date(2026, time.October, 3, 0, 0, 0, 0, time.UTC),
			HoursWorked: 8,
			Description: stringPointer("Worked on API"),
			Status:      StatusDraft,
			CreatedAt:   createdAt,
			UpdatedAt:   createdAt,
		},
	}
	service := NewService(repository)

	response, err := service.Create(context.Background(), 42, CreateInput{
		WorkDate:    "2026-10-03",
		HoursWorked: 8,
		Description: &description,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if repository.params.UserID != 42 {
		t.Fatalf("repository user ID = %d, want 42", repository.params.UserID)
	}
	if repository.params.Status != StatusDraft {
		t.Fatalf("repository status = %q, want %q", repository.params.Status, StatusDraft)
	}
	if repository.params.Description == nil || *repository.params.Description != "Worked on API" {
		t.Fatalf("repository description = %v", repository.params.Description)
	}
	if response.WorkDate != "2026-10-03" || response.UserID != 42 {
		t.Fatalf("response = %+v", response)
	}
}

func TestServiceCreateValidatesInput(t *testing.T) {
	tests := []struct {
		name   string
		userID int64
		input  CreateInput
		want   error
	}{
		{
			name:   "missing authenticated user",
			userID: 0,
			input:  CreateInput{WorkDate: "2026-10-03", HoursWorked: 8},
			want:   ErrInvalidUserID,
		},
		{
			name:   "invalid date",
			userID: 42,
			input:  CreateInput{WorkDate: "03-10-2026", HoursWorked: 8},
			want:   ErrInvalidDate,
		},
		{
			name:   "zero hours",
			userID: 42,
			input:  CreateInput{WorkDate: "2026-10-03", HoursWorked: 0},
			want:   ErrInvalidHours,
		},
		{
			name:   "too many hours",
			userID: 42,
			input:  CreateInput{WorkDate: "2026-10-03", HoursWorked: 25},
			want:   ErrInvalidHours,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &stubRepository{}
			service := NewService(repository)

			_, err := service.Create(context.Background(), test.userID, test.input)
			if !errors.Is(err, test.want) {
				t.Fatalf("Create() error = %v, want %v", err, test.want)
			}
			if repository.called {
				t.Fatal("repository was called for invalid input")
			}
		})
	}
}

func TestServiceGetAllFiltersByAuthenticatedUser(t *testing.T) {
	repository := &stubRepository{
		listResult: []Timesheet{
			{
				ID:          7,
				UserID:      42,
				WorkDate:    time.Date(2026, time.October, 3, 0, 0, 0, 0, time.UTC),
				HoursWorked: 8,
				Status:      StatusDraft,
			},
		},
	}
	service := NewService(repository)

	responses, err := service.GetAll(context.Background(), 42)
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}
	if !repository.listCalled || repository.listUserID != 42 {
		t.Fatalf("repository user ID = %d, want 42", repository.listUserID)
	}
	if len(responses) != 1 || responses[0].WorkDate != "2026-10-03" {
		t.Fatalf("responses = %+v", responses)
	}
}

func TestServiceGetAllReturnsEmptyArray(t *testing.T) {
	repository := &stubRepository{listResult: nil}
	service := NewService(repository)

	responses, err := service.GetAll(context.Background(), 42)
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}
	if responses == nil || len(responses) != 0 {
		t.Fatalf("responses = %#v, want an empty non-nil slice", responses)
	}
}

func stringPointer(value string) *string {
	return &value
}
