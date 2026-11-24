package timesheet

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"finos.com/api/internal/reimbursement"
	"github.com/jackc/pgx/v5"
)

const dateLayout = "2006-01-02"

var (
	ErrInvalidUserID     = errors.New("authenticated user is required")
	ErrInvalidDate       = errors.New("work_date must use YYYY-MM-DD format")
	ErrInvalidHours      = errors.New("hours_worked must be between 1 and 24")
	ErrNotManager        = errors.New("You are not a manager, go away")
	ErrTimesheetNotFound = errors.New("timesheet not found")
	ErrForbidden         = errors.New("only the timesheet creator can delete it")
)

type Service struct {
	repository   Repository
	receiptStore ReceiptCleaner
}

type ReceiptCleaner interface {
	Delete(ctx context.Context, asset reimbursement.ReceiptAsset) error
}

func NewService(repository Repository, receiptStore ReceiptCleaner) *Service {
	return &Service{repository: repository, receiptStore: receiptStore}
}

func (service *Service) Create(
	ctx context.Context,
	userID int64,
	input CreateInput,
) (Response, error) {
	if userID <= 0 {
		return Response{}, ErrInvalidUserID
	}

	workDate, err := time.Parse(dateLayout, input.WorkDate)
	if err != nil {
		return Response{}, ErrInvalidDate
	}
	if input.HoursWorked < 1 || input.HoursWorked > 24 {
		return Response{}, ErrInvalidHours
	}

	description := normalizeDescription(input.Description)
	createdTimesheet, err := service.repository.Create(ctx, CreateTimesheetParams{
		UserID:      userID,
		WorkDate:    workDate,
		HoursWorked: input.HoursWorked,
		Description: description,
		Status:      StatusDraft,
	})
	if err != nil {
		return Response{}, fmt.Errorf("create timesheet: %w", err)
	}

	return toResponse(createdTimesheet), nil
}

func (service *Service) GetAll(
	ctx context.Context,
	userID int64,
	userRole string,
) ([]Response, error) {
	if userID <= 0 {
		return nil, ErrInvalidUserID
	}
	var foundTimesheets []Timesheet
	var err error
	if userRole == "manager" {
		foundTimesheets, err = service.repository.GetAllTimesheet(ctx)
	} else {
		foundTimesheets, err = service.repository.FindAllByUserID(ctx, userID)
	}
	if err != nil {
		return nil, fmt.Errorf("get user timesheets: %w", err)
	}

	responses := make([]Response, 0, len(foundTimesheets))
	for _, foundTimesheet := range foundTimesheets {
		responses = append(responses, toResponse(foundTimesheet))
	}

	return responses, nil
}

// GetOptions returns timesheets available for selection controls. Employees
// receive their own timesheets, while managers receive every timesheet.
func (service *Service) GetOptions(
	ctx context.Context,
	userID int64,
	userRole string,
) ([]Option, error) {
	if userID <= 0 {
		return nil, ErrInvalidUserID
	}

	var foundTimesheets []Timesheet
	var err error
	if userRole == "manager" {
		foundTimesheets, err = service.repository.GetAllTimesheet(ctx)
	} else {
		foundTimesheets, err = service.repository.FindAllByUserID(ctx, userID)
	}
	if err != nil {
		return nil, fmt.Errorf("get timesheet options: %w", err)
	}

	options := make([]Option, 0, len(foundTimesheets))
	for _, foundTimesheet := range foundTimesheets {
		options = append(options, Option{
			ID:          foundTimesheet.ID,
			Description: foundTimesheet.Description,
		})
	}
	return options, nil
}

func (service *Service) ApproveTimesheet(
	ctx context.Context,
	timesheetID int64,
	userRole string,
) (Response, error) {
	if userRole != "manager" {
		return Response{}, ErrNotManager
	}
	approvedTimesheet, err := service.repository.ApproveTimesheet(ctx, timesheetID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Response{}, ErrTimesheetNotFound
	}
	if err != nil {
		return Response{}, fmt.Errorf("approve timesheet: %w", err)
	}
	return toResponse(approvedTimesheet), nil
}

func (service *Service) Delete(ctx context.Context, userID, timesheetID int64) error {
	if userID <= 0 {
		return ErrInvalidUserID
	}
	found, err := service.repository.FindByID(ctx, timesheetID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrTimesheetNotFound
	}
	if err != nil {
		return fmt.Errorf("find timesheet for deletion: %w", err)
	}
	if found.UserID != userID {
		return ErrForbidden
	}
	assets, err := service.repository.DeleteOwned(ctx, timesheetID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrTimesheetNotFound
	}
	if err != nil {
		return fmt.Errorf("delete timesheet: %w", err)
	}
	if len(assets) > 0 && service.receiptStore == nil {
		log.Printf("delete timesheet %d: receipt storage is not configured for cleanup", timesheetID)
		return nil
	}
	for _, asset := range assets {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		if err := service.receiptStore.Delete(cleanupCtx, asset); err != nil {
			log.Printf("delete timesheet %d: clean up receipt: %v", timesheetID, err)
		}
		cancel()
	}
	return nil
}

func normalizeDescription(description *string) *string {
	if description == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*description)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

func toResponse(timesheet Timesheet) Response {
	return Response{
		ID:          timesheet.ID,
		UserID:      timesheet.UserID,
		WorkDate:    timesheet.WorkDate.Format(dateLayout),
		HoursWorked: timesheet.HoursWorked,
		Description: timesheet.Description,
		Status:      timesheet.Status,
		CreatedAt:   timesheet.CreatedAt,
		UpdatedAt:   timesheet.UpdatedAt,
	}
}
