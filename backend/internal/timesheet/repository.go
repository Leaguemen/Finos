package timesheet

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Create(ctx context.Context, params CreateTimesheetParams) (Timesheet, error)
	FindAllByUserID(ctx context.Context, userID int64) ([]Timesheet, error)
	GetAllTimesheet(ctx context.Context) ([]Timesheet, error)
	ApproveTimesheet(ctx context.Context, timesheetID int64) (Timesheet, error)
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

func (repository *PostgresRepository) Create(
	ctx context.Context,
	params CreateTimesheetParams,
) (Timesheet, error) {
	const query = `
		INSERT INTO timesheets (user_id, work_date, hours_worked, description, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, work_date, hours_worked, description, status, created_at, updated_at
	`
	if params.Status == "" {
		params.Status = StatusDraft
	}

	createdTimesheet, err := scanTimesheet(repository.database.QueryRow(
		ctx,
		query,
		params.UserID,
		params.WorkDate,
		params.HoursWorked,
		params.Description,
		params.Status,
	))
	if err != nil {
		return Timesheet{}, fmt.Errorf("create timesheet: %w", err)
	}

	return createdTimesheet, nil
}

func (repository *PostgresRepository) FindAllByUserID(
	ctx context.Context,
	userID int64,
) ([]Timesheet, error) {
	const query = `
		SELECT id, user_id, work_date, hours_worked, description, status, created_at, updated_at
		FROM timesheets
		WHERE user_id = $1
		ORDER BY work_date DESC, id DESC
	`

	rows, err := repository.database.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query timesheets: %w", err)
	}
	defer rows.Close()

	timesheets := make([]Timesheet, 0)
	for rows.Next() {
		timesheet, err := scanTimesheet(rows)
		if err != nil {
			return nil, fmt.Errorf("scan timesheet: %w", err)
		}
		timesheets = append(timesheets, timesheet)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate timesheets: %w", err)
	}

	return timesheets, nil
}

func (repository *PostgresRepository) GetAllTimesheet(
	ctx context.Context,
) ([]Timesheet, error) {
	const query = `
		SELECT id, user_id, work_date, hours_worked, description, status, created_at, updated_at
		FROM timesheets
		ORDER BY work_date DESC, id DESC
	`
	rows, err := repository.database.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query timesheets: %w", err)
	}
	defer rows.Close()

	timesheets := make([]Timesheet, 0)
	for rows.Next() {
		timesheet, err := scanTimesheet(rows)
		if err != nil {
			return nil, fmt.Errorf("scan timesheet: %w", err)
		}
		timesheets = append(timesheets, timesheet)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate timesheets: %w", err)
	}

	return timesheets, nil
}

// ApproveTimesheet toggles approved timesheets back to draft and approves all other statuses.
func (repository *PostgresRepository) ApproveTimesheet(
	ctx context.Context,
	timesheetID int64,
) (Timesheet, error) {
	const query = `
		UPDATE timesheets
		SET status = CASE WHEN status = $2 THEN $3 ELSE $2 END,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, user_id, work_date, hours_worked, description, status, created_at, updated_at
	`

	approvedTimesheet, err := scanTimesheet(repository.database.QueryRow(
		ctx,
		query,
		timesheetID,
		StatusApproved,
		StatusDraft,
	))
	if err != nil {
		return Timesheet{}, fmt.Errorf("approve timesheet: %w", err)
	}

	return approvedTimesheet, nil
}

type rowScanner interface {
	Scan(destinations ...any) error
}

func scanTimesheet(row rowScanner) (Timesheet, error) {
	var timesheet Timesheet

	err := row.Scan(
		&timesheet.ID,
		&timesheet.UserID,
		&timesheet.WorkDate,
		&timesheet.HoursWorked,
		&timesheet.Description,
		&timesheet.Status,
		&timesheet.CreatedAt,
		&timesheet.UpdatedAt,
	)
	if err != nil {
		return Timesheet{}, err
	}

	return timesheet, nil
}
