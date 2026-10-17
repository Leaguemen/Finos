package reimbursement

import (
	"io"
	"time"
)

type Status string

const (
	StatusDraft     Status = "draft"
	StatusSubmitted Status = "submitted"
	StatusApproved  Status = "approved"
	StatusRejected  Status = "rejected"
)

type ReceiptAsset struct {
	PublicID     string
	AssetID      string
	Format       string
	ResourceType string
	DeliveryType string
}

type Reimbursement struct {
	ID          int64
	UserID      int64
	TimesheetID int64
	ExpenseDate time.Time
	Amount      string
	Category    string
	Description *string
	Receipt     *ReceiptAsset
	Status      Status
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CreateParams struct {
	UserID      int64
	TimesheetID int64
	ExpenseDate time.Time
	Amount      string
	Category    string
	Description *string
	Receipt     *ReceiptAsset
	Status      Status
}

type ReceiptUpload struct {
	Reader      io.Reader
	ContentType string
}

type CreateInput struct {
	ExpenseDate string
	Amount      string
	Category    string
	Description *string
	Receipt     *ReceiptUpload
}

type Response struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	TimesheetID int64     `json:"timesheet_id"`
	ExpenseDate string    `json:"expense_date"`
	Amount      string    `json:"amount"`
	Category    string    `json:"category"`
	Description *string   `json:"description"`
	Status      Status    `json:"status"`
	HasReceipt  bool      `json:"has_receipt"`
	ReceiptURL  *string   `json:"receipt_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
