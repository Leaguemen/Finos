package reimbursement

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"
)

type stubRepository struct {
	belongs          bool
	timesheetExists  bool
	created          Reimbursement
	found            Reimbursement
	list             []Reimbursement
	err              error
	createParams     CreateParams
	createCalled     bool
	ownershipChecked bool
}

func (repository *stubRepository) TimesheetExists(context.Context, int64) (bool, error) {
	return repository.timesheetExists, repository.err
}

func (repository *stubRepository) TimesheetBelongsToUser(context.Context, int64, int64) (bool, error) {
	repository.ownershipChecked = true
	return repository.belongs, repository.err
}

func (repository *stubRepository) Create(_ context.Context, params CreateParams) (Reimbursement, error) {
	repository.createCalled = true
	repository.createParams = params
	return repository.created, repository.err
}

func (repository *stubRepository) ListByTimesheet(context.Context, int64) ([]Reimbursement, error) {
	return repository.list, repository.err
}

func (repository *stubRepository) FindByID(context.Context, int64) (Reimbursement, error) {
	return repository.found, repository.err
}

type stubReceiptStore struct {
	asset        ReceiptAsset
	uploadCalled bool
	deleteCalled bool
	downloadURL  string
	err          error
}

func (store *stubReceiptStore) Upload(context.Context, ReceiptUpload) (ReceiptAsset, error) {
	store.uploadCalled = true
	return store.asset, store.err
}

func (store *stubReceiptStore) Delete(context.Context, ReceiptAsset) error {
	store.deleteCalled = true
	return store.err
}

func (store *stubReceiptStore) DownloadURL(ReceiptAsset, time.Time) (string, error) {
	return store.downloadURL, store.err
}

func TestCreateUsesAuthenticatedUserAndUploadsReceipt(t *testing.T) {
	asset := ReceiptAsset{
		PublicID: "finos/receipts/abc", AssetID: "asset", Format: "png",
		ResourceType: "image", DeliveryType: "authenticated",
	}
	repository := &stubRepository{
		belongs: true,
		created: Reimbursement{
			ID: 23, UserID: 42, TimesheetID: 12,
			ExpenseDate: time.Date(2026, time.October, 17, 0, 0, 0, 0, time.UTC),
			Amount:      "125000.00", Category: "transportation", Receipt: &asset,
			Status: StatusDraft,
		},
	}
	store := &stubReceiptStore{asset: asset}
	service := NewService(repository, store)

	response, err := service.Create(context.Background(), 42, 12, CreateInput{
		ExpenseDate: "2026-10-17",
		Amount:      "125000",
		Category:    " transportation ",
		Receipt: &ReceiptUpload{
			Reader:      bytes.NewReader([]byte("image")),
			ContentType: "image/png",
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if !repository.ownershipChecked || !repository.createCalled || !store.uploadCalled {
		t.Fatal("expected ownership check, receipt upload, and repository creation")
	}
	if repository.createParams.UserID != 42 || repository.createParams.TimesheetID != 12 {
		t.Fatalf("create params = %+v", repository.createParams)
	}
	if repository.createParams.Amount != "125000.00" || repository.createParams.Category != "transportation" {
		t.Fatalf("normalized params = %+v", repository.createParams)
	}
	if repository.createParams.Status != StatusDraft || repository.createParams.Receipt == nil {
		t.Fatalf("create params = %+v", repository.createParams)
	}
	if !response.HasReceipt || response.ReceiptURL == nil || *response.ReceiptURL != "/api/v1/reimbursements/23/receipt" {
		t.Fatalf("response = %+v", response)
	}
}

func TestCreateRejectsAnotherUsersTimesheetBeforeUpload(t *testing.T) {
	repository := &stubRepository{belongs: false}
	store := &stubReceiptStore{}
	service := NewService(repository, store)

	_, err := service.Create(context.Background(), 42, 12, CreateInput{
		ExpenseDate: "2026-10-17", Amount: "10.00", Category: "meal",
		Receipt: &ReceiptUpload{Reader: bytes.NewReader([]byte("image"))},
	})
	if !errors.Is(err, ErrTimesheetNotFound) {
		t.Fatalf("Create() error = %v, want ErrTimesheetNotFound", err)
	}
	if store.uploadCalled || repository.createCalled {
		t.Fatal("unauthorized creation uploaded a receipt or wrote to the repository")
	}
}

func TestCreateDeletesUploadedReceiptWhenDatabaseWriteFails(t *testing.T) {
	repository := &stubRepository{belongs: true, err: errors.New("database unavailable")}
	store := &stubReceiptStore{asset: ReceiptAsset{PublicID: "receipt"}}
	service := NewService(repository, store)

	_, err := service.Create(context.Background(), 42, 12, CreateInput{
		ExpenseDate: "2026-10-17", Amount: "10.00", Category: "meal",
		Receipt: &ReceiptUpload{Reader: bytes.NewReader([]byte("image"))},
	})
	if err == nil {
		t.Fatal("Create() error = nil")
	}
	if store.uploadCalled {
		t.Fatal("ownership lookup error should prevent upload")
	}

	repository.err = nil
	repository.created = Reimbursement{}
	store.err = nil
	// Make only the create call fail by using a repository wrapper.
	failing := &createFailureRepository{stubRepository: repository}
	service = NewService(failing, store)
	_, err = service.Create(context.Background(), 42, 12, CreateInput{
		ExpenseDate: "2026-10-17", Amount: "10.00", Category: "meal",
		Receipt: &ReceiptUpload{Reader: bytes.NewReader([]byte("image"))},
	})
	if err == nil || !store.deleteCalled {
		t.Fatalf("error = %v, receipt deleted = %t", err, store.deleteCalled)
	}
}

type createFailureRepository struct {
	*stubRepository
}

func (repository *createFailureRepository) Create(context.Context, CreateParams) (Reimbursement, error) {
	return Reimbursement{}, errors.New("database unavailable")
}

func TestReimbursementAccessRules(t *testing.T) {
	reimbursement := Reimbursement{ID: 23, UserID: 42, TimesheetID: 12}

	t.Run("owner can get", func(t *testing.T) {
		service := NewService(&stubRepository{found: reimbursement}, &stubReceiptStore{})
		if _, err := service.Get(context.Background(), 42, "employee", 23); err != nil {
			t.Fatalf("Get() error = %v", err)
		}
	})

	t.Run("other employee denied", func(t *testing.T) {
		service := NewService(&stubRepository{found: reimbursement}, &stubReceiptStore{})
		_, err := service.Get(context.Background(), 99, "employee", 23)
		if !errors.Is(err, ErrForbidden) {
			t.Fatalf("Get() error = %v, want ErrForbidden", err)
		}
	})

	t.Run("manager can get", func(t *testing.T) {
		service := NewService(&stubRepository{found: reimbursement}, &stubReceiptStore{})
		if _, err := service.Get(context.Background(), 99, "manager", 23); err != nil {
			t.Fatalf("Get() error = %v", err)
		}
	})
}

func TestListReturnsEmptyArray(t *testing.T) {
	repository := &stubRepository{belongs: true, list: nil}
	service := NewService(repository, &stubReceiptStore{})

	responses, err := service.ListByTimesheet(context.Background(), 42, "employee", 12)
	if err != nil {
		t.Fatalf("ListByTimesheet() error = %v", err)
	}
	if responses == nil || len(responses) != 0 {
		t.Fatalf("responses = %#v, want empty non-nil slice", responses)
	}
}
