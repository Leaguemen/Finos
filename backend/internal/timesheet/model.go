package timesheet

import "time"

type Status string

const (
	StatusDraft     Status = "draft"
	StatusSubmitted Status = "submitted"
	StatusApproved  Status = "approved"
	StatusRejected  Status = "rejected"
)

type Timesheet struct {
	ID          int64
	UserID      int64
	WorkDate    time.Time
	HoursWorked int32
	Description *string
	Status      Status
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CreateTimesheetParams struct {
	UserID      int64
	WorkDate    time.Time
	HoursWorked int32
	Description *string
	Status      Status
}

type CreateInput struct {
	WorkDate    string  `json:"work_date"`
	HoursWorked int32   `json:"hours_worked"`
	Description *string `json:"description"`
}

type Response struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	WorkDate    string    `json:"work_date"`
	HoursWorked int32     `json:"hours_worked"`
	Description *string   `json:"description"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
