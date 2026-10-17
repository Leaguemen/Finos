package reimbursement

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("reimbursement not found")

type Repository interface {
	TimesheetExists(ctx context.Context, timesheetID int64) (bool, error)
	TimesheetBelongsToUser(ctx context.Context, timesheetID, userID int64) (bool, error)
	Create(ctx context.Context, params CreateParams) (Reimbursement, error)
	ListByTimesheet(ctx context.Context, timesheetID int64) ([]Reimbursement, error)
	FindByID(ctx context.Context, reimbursementID int64) (Reimbursement, error)
}

func (repository *PostgresRepository) TimesheetExists(
	ctx context.Context,
	timesheetID int64,
) (bool, error) {
	const query = `SELECT EXISTS (SELECT 1 FROM timesheets WHERE id = $1)`

	var exists bool
	if err := repository.database.QueryRow(ctx, query, timesheetID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check timesheet existence: %w", err)
	}
	return exists, nil
}

type database interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type PostgresRepository struct {
	database database
}

func NewRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{database: pool}
}

func (repository *PostgresRepository) TimesheetBelongsToUser(
	ctx context.Context,
	timesheetID int64,
	userID int64,
) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM timesheets
			WHERE id = $1 AND user_id = $2
		)
	`

	var belongs bool
	if err := repository.database.QueryRow(ctx, query, timesheetID, userID).Scan(&belongs); err != nil {
		return false, fmt.Errorf("check timesheet ownership: %w", err)
	}
	return belongs, nil
}

func (repository *PostgresRepository) Create(
	ctx context.Context,
	params CreateParams,
) (Reimbursement, error) {
	const query = `
		INSERT INTO reimbursements (
			user_id, timesheet_id, expense_date, amount, category, description,
			receipt_public_id, receipt_asset_id, receipt_format,
			receipt_resource_type, receipt_delivery_type, status
		)
		SELECT
			timesheets.user_id, timesheets.id, $3, $4::numeric, $5, $6,
			$7, $8, $9, $10, $11, $12
		FROM timesheets
		WHERE timesheets.id = $1 AND timesheets.user_id = $2
		RETURNING
			id, user_id, timesheet_id, expense_date, amount::text, category,
			description, receipt_public_id, receipt_asset_id, receipt_format,
			receipt_resource_type, receipt_delivery_type, status, created_at, updated_at
	`

	var publicID, assetID, format, resourceType, deliveryType *string
	if params.Receipt != nil {
		publicID = &params.Receipt.PublicID
		assetID = &params.Receipt.AssetID
		format = &params.Receipt.Format
		resourceType = &params.Receipt.ResourceType
		deliveryType = &params.Receipt.DeliveryType
	}

	created, err := scanReimbursement(repository.database.QueryRow(
		ctx,
		query,
		params.TimesheetID,
		params.UserID,
		params.ExpenseDate,
		params.Amount,
		params.Category,
		params.Description,
		publicID,
		assetID,
		format,
		resourceType,
		deliveryType,
		params.Status,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return Reimbursement{}, ErrNotFound
	}
	if err != nil {
		return Reimbursement{}, fmt.Errorf("create reimbursement: %w", err)
	}
	return created, nil
}

func (repository *PostgresRepository) ListByTimesheet(
	ctx context.Context,
	timesheetID int64,
) ([]Reimbursement, error) {
	const query = `
		SELECT
			id, user_id, timesheet_id, expense_date, amount::text, category,
			description, receipt_public_id, receipt_asset_id, receipt_format,
			receipt_resource_type, receipt_delivery_type, status, created_at, updated_at
		FROM reimbursements
		WHERE timesheet_id = $1
		ORDER BY expense_date DESC, id DESC
	`

	rows, err := repository.database.Query(ctx, query, timesheetID)
	if err != nil {
		return nil, fmt.Errorf("list reimbursements: %w", err)
	}
	defer rows.Close()

	reimbursements := make([]Reimbursement, 0)
	for rows.Next() {
		reimbursement, err := scanReimbursement(rows)
		if err != nil {
			return nil, fmt.Errorf("scan reimbursement: %w", err)
		}
		reimbursements = append(reimbursements, reimbursement)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate reimbursements: %w", err)
	}
	return reimbursements, nil
}

func (repository *PostgresRepository) FindByID(
	ctx context.Context,
	reimbursementID int64,
) (Reimbursement, error) {
	const query = `
		SELECT
			id, user_id, timesheet_id, expense_date, amount::text, category,
			description, receipt_public_id, receipt_asset_id, receipt_format,
			receipt_resource_type, receipt_delivery_type, status, created_at, updated_at
		FROM reimbursements
		WHERE id = $1
	`

	found, err := scanReimbursement(repository.database.QueryRow(ctx, query, reimbursementID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Reimbursement{}, ErrNotFound
	}
	if err != nil {
		return Reimbursement{}, fmt.Errorf("find reimbursement: %w", err)
	}
	return found, nil
}

type rowScanner interface {
	Scan(destinations ...any) error
}

func scanReimbursement(row rowScanner) (Reimbursement, error) {
	var reimbursement Reimbursement
	var publicID, assetID, format, resourceType, deliveryType *string

	err := row.Scan(
		&reimbursement.ID,
		&reimbursement.UserID,
		&reimbursement.TimesheetID,
		&reimbursement.ExpenseDate,
		&reimbursement.Amount,
		&reimbursement.Category,
		&reimbursement.Description,
		&publicID,
		&assetID,
		&format,
		&resourceType,
		&deliveryType,
		&reimbursement.Status,
		&reimbursement.CreatedAt,
		&reimbursement.UpdatedAt,
	)
	if err != nil {
		return Reimbursement{}, err
	}

	if publicID != nil {
		reimbursement.Receipt = &ReceiptAsset{
			PublicID:     *publicID,
			AssetID:      valueOrEmpty(assetID),
			Format:       valueOrEmpty(format),
			ResourceType: valueOrEmpty(resourceType),
			DeliveryType: valueOrEmpty(deliveryType),
		}
	}
	return reimbursement, nil
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
