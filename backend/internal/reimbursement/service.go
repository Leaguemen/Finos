package reimbursement

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"
)

const (
	dateLayout              = "2006-01-02"
	maximumCategoryRunes    = 100
	maximumDescriptionRunes = 1000
	receiptURLLifetime      = 5 * time.Minute
)

var (
	ErrInvalidUserID      = errors.New("authenticated user is required")
	ErrInvalidTimesheetID = errors.New("timesheet ID must be a positive integer")
	ErrTimesheetNotFound  = errors.New("timesheet not found")
	ErrInvalidDate        = errors.New("expense_date must use YYYY-MM-DD format")
	ErrInvalidAmount      = errors.New("amount must be greater than zero, use at most 10 integer digits, and have at most 2 decimal places")
	ErrCategoryRequired   = errors.New("category is required")
	ErrCategoryTooLong    = errors.New("category must not exceed 100 characters")
	ErrDescriptionTooLong = errors.New("description must not exceed 1000 characters")
	ErrForbidden          = errors.New("reimbursement access is forbidden")
	ErrReceiptNotFound    = errors.New("receipt not found")
	amountPattern         = regexp.MustCompile(`^(0|[1-9][0-9]{0,9})(\.[0-9]{1,2})?$`)
	maximumAmount, _      = new(big.Rat).SetString("9999999999.99")
)

type ReceiptStore interface {
	Upload(ctx context.Context, receipt ReceiptUpload) (ReceiptAsset, error)
	Delete(ctx context.Context, asset ReceiptAsset) error
	DownloadURL(asset ReceiptAsset, expiresAt time.Time) (string, error)
}

type Service struct {
	repository   Repository
	receiptStore ReceiptStore
	clock        func() time.Time
}

func NewService(repository Repository, receiptStore ReceiptStore) *Service {
	return &Service{
		repository:   repository,
		receiptStore: receiptStore,
		clock:        time.Now,
	}
}

func (service *Service) Create(
	ctx context.Context,
	userID int64,
	timesheetID int64,
	input CreateInput,
) (Response, error) {
	if userID <= 0 {
		return Response{}, ErrInvalidUserID
	}
	if timesheetID <= 0 {
		return Response{}, ErrInvalidTimesheetID
	}

	expenseDate, amount, category, description, err := validateCreateInput(input)
	if err != nil {
		return Response{}, err
	}

	belongs, err := service.repository.TimesheetBelongsToUser(ctx, timesheetID, userID)
	if err != nil {
		return Response{}, fmt.Errorf("authorize reimbursement creation: %w", err)
	}
	if !belongs {
		return Response{}, ErrTimesheetNotFound
	}

	var receipt *ReceiptAsset
	if input.Receipt != nil {
		if service.receiptStore == nil {
			return Response{}, errors.New("receipt storage is not configured")
		}
		uploaded, err := service.receiptStore.Upload(ctx, *input.Receipt)
		if err != nil {
			return Response{}, fmt.Errorf("upload receipt: %w", err)
		}
		receipt = &uploaded
	}

	created, err := service.repository.Create(ctx, CreateParams{
		UserID:      userID,
		TimesheetID: timesheetID,
		ExpenseDate: expenseDate,
		Amount:      amount,
		Category:    category,
		Description: description,
		Receipt:     receipt,
		Status:      StatusDraft,
	})
	if err != nil {
		if receipt != nil {
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
			defer cancel()
			if cleanupErr := service.receiptStore.Delete(cleanupCtx, *receipt); cleanupErr != nil {
				return Response{}, fmt.Errorf("create reimbursement: %w; clean up receipt: %v", err, cleanupErr)
			}
		}
		if errors.Is(err, ErrNotFound) {
			return Response{}, ErrTimesheetNotFound
		}
		return Response{}, fmt.Errorf("create reimbursement: %w", err)
	}

	return toResponse(created), nil
}

func (service *Service) ListByTimesheet(
	ctx context.Context,
	userID int64,
	userRole string,
	timesheetID int64,
) ([]Response, error) {
	if userID <= 0 {
		return nil, ErrInvalidUserID
	}
	if timesheetID <= 0 {
		return nil, ErrInvalidTimesheetID
	}
	if err := service.authorizeTimesheet(ctx, userID, userRole, timesheetID); err != nil {
		return nil, err
	}

	found, err := service.repository.ListByTimesheet(ctx, timesheetID)
	if err != nil {
		return nil, fmt.Errorf("list reimbursements: %w", err)
	}
	responses := make([]Response, 0, len(found))
	for _, reimbursement := range found {
		responses = append(responses, toResponse(reimbursement))
	}
	return responses, nil
}

func (service *Service) Get(
	ctx context.Context,
	userID int64,
	userRole string,
	reimbursementID int64,
) (Response, error) {
	found, err := service.findAuthorized(ctx, userID, userRole, reimbursementID)
	if err != nil {
		return Response{}, err
	}
	return toResponse(found), nil
}

func (service *Service) ReceiptURL(
	ctx context.Context,
	userID int64,
	userRole string,
	reimbursementID int64,
) (string, error) {
	found, err := service.findAuthorized(ctx, userID, userRole, reimbursementID)
	if err != nil {
		return "", err
	}
	if found.Receipt == nil {
		return "", ErrReceiptNotFound
	}
	if service.receiptStore == nil {
		return "", errors.New("receipt storage is not configured")
	}

	url, err := service.receiptStore.DownloadURL(
		*found.Receipt,
		service.clock().UTC().Add(receiptURLLifetime),
	)
	if err != nil {
		return "", fmt.Errorf("generate receipt URL: %w", err)
	}
	return url, nil
}

func (service *Service) authorizeTimesheet(
	ctx context.Context,
	userID int64,
	userRole string,
	timesheetID int64,
) error {
	if userRole == "manager" {
		exists, err := service.repository.TimesheetExists(ctx, timesheetID)
		if err != nil {
			return fmt.Errorf("authorize timesheet access: %w", err)
		}
		if !exists {
			return ErrTimesheetNotFound
		}
		return nil
	}
	belongs, err := service.repository.TimesheetBelongsToUser(ctx, timesheetID, userID)
	if err != nil {
		return fmt.Errorf("authorize timesheet access: %w", err)
	}
	if !belongs {
		return ErrForbidden
	}
	return nil
}

func (service *Service) findAuthorized(
	ctx context.Context,
	userID int64,
	userRole string,
	reimbursementID int64,
) (Reimbursement, error) {
	if userID <= 0 {
		return Reimbursement{}, ErrInvalidUserID
	}
	if reimbursementID <= 0 {
		return Reimbursement{}, ErrNotFound
	}
	found, err := service.repository.FindByID(ctx, reimbursementID)
	if err != nil {
		return Reimbursement{}, err
	}
	if userRole != "manager" && found.UserID != userID {
		return Reimbursement{}, ErrForbidden
	}
	return found, nil
}

func validateCreateInput(input CreateInput) (time.Time, string, string, *string, error) {
	expenseDate, err := time.Parse(dateLayout, input.ExpenseDate)
	if err != nil {
		return time.Time{}, "", "", nil, ErrInvalidDate
	}

	amountText := strings.TrimSpace(input.Amount)
	if !amountPattern.MatchString(amountText) {
		return time.Time{}, "", "", nil, ErrInvalidAmount
	}
	amount, ok := new(big.Rat).SetString(amountText)
	if !ok || amount.Sign() <= 0 || amount.Cmp(maximumAmount) > 0 {
		return time.Time{}, "", "", nil, ErrInvalidAmount
	}

	category := strings.TrimSpace(input.Category)
	if category == "" {
		return time.Time{}, "", "", nil, ErrCategoryRequired
	}
	if len([]rune(category)) > maximumCategoryRunes {
		return time.Time{}, "", "", nil, ErrCategoryTooLong
	}

	var description *string
	if input.Description != nil {
		trimmed := strings.TrimSpace(*input.Description)
		if len([]rune(trimmed)) > maximumDescriptionRunes {
			return time.Time{}, "", "", nil, ErrDescriptionTooLong
		}
		if trimmed != "" {
			description = &trimmed
		}
	}

	return expenseDate, amount.FloatString(2), category, description, nil
}

func toResponse(reimbursement Reimbursement) Response {
	response := Response{
		ID:          reimbursement.ID,
		UserID:      reimbursement.UserID,
		TimesheetID: reimbursement.TimesheetID,
		ExpenseDate: reimbursement.ExpenseDate.Format(dateLayout),
		Amount:      reimbursement.Amount,
		Category:    reimbursement.Category,
		Description: reimbursement.Description,
		Status:      reimbursement.Status,
		HasReceipt:  reimbursement.Receipt != nil,
		CreatedAt:   reimbursement.CreatedAt,
		UpdatedAt:   reimbursement.UpdatedAt,
	}
	if reimbursement.Receipt != nil {
		url := fmt.Sprintf("/api/v1/reimbursements/%d/receipt", reimbursement.ID)
		response.ReceiptURL = &url
	}
	return response
}
