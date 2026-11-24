package timesheet

import (
	"context"
	"errors"
	"testing"
	"time"

	"finos.com/api/internal/reimbursement"
)

type stubRepository struct {
	params        CreateTimesheetParams
	result        Timesheet
	listResult    []Timesheet
	err           error
	called        bool
	listUserID    int64
	listCalled    bool
	approveCalled bool
	approvedID    int64
	deletedID     int64
	deletedUserID int64
	deleteCalled  bool
	assets        []reimbursement.ReceiptAsset
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

func (repository *stubRepository) GetAllTimesheet(_ context.Context) ([]Timesheet, error) {
	repository.listCalled = true
	return repository.listResult, repository.err
}

func (repository *stubRepository) ApproveTimesheet(_ context.Context, timesheetID int64) (Timesheet, error) {
	repository.approveCalled = true
	repository.approvedID = timesheetID
	return repository.result, repository.err
}

func (repository *stubRepository) FindByID(context.Context, int64) (Timesheet, error) {
	return repository.result, repository.err
}

func (repository *stubRepository) DeleteOwned(_ context.Context, timesheetID, userID int64) ([]reimbursement.ReceiptAsset, error) {
	repository.deleteCalled = true
	repository.deletedID = timesheetID
	repository.deletedUserID = userID
	return repository.assets, repository.err
}

type stubReceiptCleaner struct {
	deleted []reimbursement.ReceiptAsset
}

func (cleaner *stubReceiptCleaner) Delete(_ context.Context, asset reimbursement.ReceiptAsset) error {
	cleaner.deleted = append(cleaner.deleted, asset)
	return nil
}

func TestDeleteTimesheetRequiresCreatorAndCleansCascadedReceipts(t *testing.T) {
	asset := reimbursement.ReceiptAsset{PublicID: "receipt-1"}
	repository := &stubRepository{result: Timesheet{ID: 7, UserID: 42}, assets: []reimbursement.ReceiptAsset{asset}}
	cleaner := &stubReceiptCleaner{}
	service := NewService(repository, cleaner)
	if err := service.Delete(context.Background(), 99, 7); !errors.Is(err, ErrForbidden) {
		t.Fatalf("other user error = %v, want forbidden", err)
	}
	if repository.deleteCalled {
		t.Fatal("deleted another user's timesheet")
	}
	if err := service.Delete(context.Background(), 42, 7); err != nil {
		t.Fatalf("owner delete error = %v", err)
	}
	if !repository.deleteCalled || repository.deletedID != 7 || repository.deletedUserID != 42 {
		t.Fatalf("delete call = %+v", repository)
	}
	if len(cleaner.deleted) != 1 || cleaner.deleted[0] != asset {
		t.Fatalf("cleaned receipts = %+v", cleaner.deleted)
	}
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
	service := NewService(repository, nil)

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
			service := NewService(repository, nil)

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
	service := NewService(repository, nil)

	responses, err := service.GetAll(context.Background(), 42, "employee")
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
	service := NewService(repository, nil)

	responses, err := service.GetAll(context.Background(), 42, "employee")
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}
	if responses == nil || len(responses) != 0 {
		t.Fatalf("responses = %#v, want an empty non-nil slice", responses)
	}
}

func TestServiceGetOptionsEmployeeUsesAuthenticatedUser(t *testing.T) {
	description := "October work"
	repository := &stubRepository{
		listResult: []Timesheet{{ID: 7, UserID: 42, Description: &description}},
	}
	service := NewService(repository, nil)

	options, err := service.GetOptions(context.Background(), 42, "employee")
	if err != nil {
		t.Fatalf("GetOptions() error = %v", err)
	}
	if repository.listUserID != 42 {
		t.Fatalf("repository user ID = %d, want 42", repository.listUserID)
	}
	if len(options) != 1 || options[0].ID != 7 || options[0].Description == nil || *options[0].Description != description {
		t.Fatalf("options = %+v", options)
	}
}

func TestServiceGetOptionsManagerGetsAllTimesheets(t *testing.T) {
	description := "Employee work"
	repository := &stubRepository{
		listResult: []Timesheet{{ID: 7, UserID: 99, Description: &description}},
	}
	service := NewService(repository, nil)

	options, err := service.GetOptions(context.Background(), 42, "manager")
	if err != nil {
		t.Fatalf("GetOptions() error = %v", err)
	}
	if repository.listUserID != 0 {
		t.Fatalf("manager options were filtered by user ID %d", repository.listUserID)
	}
	if len(options) != 1 || options[0].ID != 7 {
		t.Fatalf("options = %+v", options)
	}
}

func TestServiceGetOptionsReturnsEmptyArray(t *testing.T) {
	repository := &stubRepository{listResult: nil}
	service := NewService(repository, nil)

	options, err := service.GetOptions(context.Background(), 42, "employee")
	if err != nil {
		t.Fatalf("GetOptions() error = %v", err)
	}
	if options == nil || len(options) != 0 {
		t.Fatalf("options = %#v, want empty non-nil slice", options)
	}
}

func stringPointer(value string) *string {
	return &value
}
