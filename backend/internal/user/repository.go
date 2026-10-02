package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound   = errors.New("user not found")
	ErrEmailTaken = errors.New("email already registered")
)

type Repository interface {
	Create(ctx context.Context, params CreateParams) (User, error)
	FindByEmail(ctx context.Context, email string) (User, error)
	FindByID(ctx context.Context, id int64) (User, error)
}

type queryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type PostgresRepository struct {
	database queryRower
}

func NewRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{database: pool}
}

func (repository *PostgresRepository) Create(
	ctx context.Context,
	params CreateParams,
) (User, error) {
	const query = `
		INSERT INTO users (name, email, password_hash, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, email, password_hash, role, created_at, updated_at
	`

	if params.Role == "" {
		params.Role = RoleEmployee
	}
	params.Email = normalizeEmail(params.Email)

	createdUser, err := scanUser(repository.database.QueryRow(
		ctx,
		query,
		params.Name,
		params.Email,
		params.PasswordHash,
		params.Role,
	))
	if err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) && postgresError.Code == "23505" {
			return User{}, ErrEmailTaken
		}

		return User{}, fmt.Errorf("create user: %w", err)
	}

	return createdUser, nil
}

func (repository *PostgresRepository) FindByEmail(
	ctx context.Context,
	email string,
) (User, error) {
	const query = `
		SELECT id, name, email, password_hash, role, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	foundUser, err := scanUser(repository.database.QueryRow(
		ctx,
		query,
		normalizeEmail(email),
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("find user by email: %w", err)
	}

	return foundUser, nil
}

func (repository *PostgresRepository) FindByID(
	ctx context.Context,
	id int64,
) (User, error) {
	const query = `
		SELECT id, name, email, password_hash, role, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	foundUser, err := scanUser(repository.database.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("find user by ID: %w", err)
	}

	return foundUser, nil
}

func scanUser(row pgx.Row) (User, error) {
	var foundUser User

	err := row.Scan(
		&foundUser.ID,
		&foundUser.Name,
		&foundUser.Email,
		&foundUser.PasswordHash,
		&foundUser.Role,
		&foundUser.CreatedAt,
		&foundUser.UpdatedAt,
	)
	if err != nil {
		return User{}, err
	}

	return foundUser, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
