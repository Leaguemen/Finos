package timesheet

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const dateLayout = "2006-01-02"

var (
	ErrInvalidUserID     = errors.New("authenticated user is required")
	ErrInvalidDate       = errors.New("work_date must use YYYY-MM-DD format")
	ErrInvalidHours      = errors.New("hours_worked must be between 1 and 24")
	ErrNotManager        = errors.New("You are not a manager, go away")
	ErrTimesheetNotFound = errors.New("timesheet not found")
)

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
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
