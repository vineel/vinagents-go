package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRefreshTokenNotFound = errors.New("refresh token not found")

type RefreshTokenRepository struct {
	pool *pgxpool.Pool
}

func NewRefreshTokenRepository(pool *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{pool: pool}
}

func (r *RefreshTokenRepository) FindByToken(ctx context.Context, token string) (*RefreshToken, error) {
	query := `
		SELECT refresh_token_id, token, user_id, expires_at, created_at
		FROM app.refresh_tokens
		WHERE token = $1
	`
	return r.scanToken(r.pool.QueryRow(ctx, query, token))
}

func (r *RefreshTokenRepository) FindByUserID(ctx context.Context, userID string) ([]RefreshToken, error) {
	query := `
		SELECT refresh_token_id, token, user_id, expires_at, created_at
		FROM app.refresh_tokens
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []RefreshToken
	for rows.Next() {
		token, err := r.scanTokenFromRows(rows)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, *token)
	}

	return tokens, rows.Err()
}

func (r *RefreshTokenRepository) Create(ctx context.Context, input CreateRefreshTokenInput) (*RefreshToken, error) {
	query := `
		INSERT INTO app.refresh_tokens (token, user_id, expires_at)
		VALUES ($1, $2, $3)
		RETURNING refresh_token_id, token, user_id, expires_at, created_at
	`
	return r.scanToken(r.pool.QueryRow(ctx, query, input.Token, input.UserID, input.ExpiresAt))
}

func (r *RefreshTokenRepository) Delete(ctx context.Context, token string) error {
	query := `DELETE FROM app.refresh_tokens WHERE token = $1`
	result, err := r.pool.Exec(ctx, query, token)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrRefreshTokenNotFound
	}
	return nil
}

func (r *RefreshTokenRepository) DeleteByUserID(ctx context.Context, userID string) (int64, error) {
	query := `DELETE FROM app.refresh_tokens WHERE user_id = $1`
	result, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

func (r *RefreshTokenRepository) DeleteExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM app.refresh_tokens WHERE expires_at < NOW()`
	result, err := r.pool.Exec(ctx, query)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

func (r *RefreshTokenRepository) scanToken(row pgx.Row) (*RefreshToken, error) {
	var token RefreshToken
	err := row.Scan(
		&token.RefreshTokenID,
		&token.Token,
		&token.UserID,
		&token.ExpiresAt,
		&token.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRefreshTokenNotFound
		}
		return nil, err
	}
	return &token, nil
}

func (r *RefreshTokenRepository) scanTokenFromRows(rows pgx.Rows) (*RefreshToken, error) {
	var token RefreshToken
	err := rows.Scan(
		&token.RefreshTokenID,
		&token.Token,
		&token.UserID,
		&token.ExpiresAt,
		&token.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &token, nil
}
