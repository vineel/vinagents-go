package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("user not found")

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*User, error) {
	query := `
		SELECT user_id, email, password, first_name, last_name, is_active, created_at, updated_at
		FROM app.users
		WHERE user_id = $1
	`
	return r.scanUser(r.pool.QueryRow(ctx, query, id))
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT user_id, email, password, first_name, last_name, is_active, created_at, updated_at
		FROM app.users
		WHERE email = $1
	`
	return r.scanUser(r.pool.QueryRow(ctx, query, email))
}

func (r *UserRepository) FindAll(ctx context.Context, limit, offset int) ([]User, error) {
	query := `
		SELECT user_id, email, password, first_name, last_name, is_active, created_at, updated_at
		FROM app.users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		user, err := r.scanUserFromRows(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *user)
	}

	return users, rows.Err()
}

func (r *UserRepository) Create(ctx context.Context, input CreateUserInput) (*User, error) {
	query := `
		INSERT INTO app.users (email, password, first_name, last_name)
		VALUES ($1, $2, $3, $4)
		RETURNING user_id, email, password, first_name, last_name, is_active, created_at, updated_at
	`
	return r.scanUser(r.pool.QueryRow(ctx, query, input.Email, input.Password, input.FirstName, input.LastName))
}

func (r *UserRepository) Update(ctx context.Context, id string, input UpdateUserInput) (*User, error) {
	// Build dynamic update query
	query := `
		UPDATE app.users
		SET
			email = COALESCE($2, email),
			password = COALESCE($3, password),
			first_name = COALESCE($4, first_name),
			last_name = COALESCE($5, last_name),
			is_active = COALESCE($6, is_active),
			updated_at = NOW()
		WHERE user_id = $1
		RETURNING user_id, email, password, first_name, last_name, is_active, created_at, updated_at
	`
	return r.scanUser(r.pool.QueryRow(ctx, query, id, input.Email, input.Password, input.FirstName, input.LastName, input.IsActive))
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM app.users WHERE user_id = $1`
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *UserRepository) Exists(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM app.users WHERE email = $1)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, email).Scan(&exists)
	return exists, err
}

func (r *UserRepository) scanUser(row pgx.Row) (*User, error) {
	var user User
	err := row.Scan(
		&user.UserID,
		&user.Email,
		&user.Password,
		&user.FirstName,
		&user.LastName,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) scanUserFromRows(rows pgx.Rows) (*User, error) {
	var user User
	err := rows.Scan(
		&user.UserID,
		&user.Email,
		&user.Password,
		&user.FirstName,
		&user.LastName,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
