package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/uddinArsalan/ferry-registry/domain"
)

type AuthRepo struct {
	db *sql.DB
}

func NewAuthRepo(db *sql.DB) *AuthRepo {
	return &AuthRepo{db: db}
}

func (a *AuthRepo) CreateUser(ctx context.Context, name, email, passwordHash string) (int64, error) {
	var userID int64
	query := "INSERT INTO users (name, email, password_hash) VALUES ($1, $2, $3) returning id"
	row := a.db.QueryRow(query, name, email, passwordHash)
	if err := row.Scan(&userID); err != nil {
		return -1, err
	}
	return userID, nil
}

func (a *AuthRepo) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	var user domain.User
	query := "SELECT id, name, email, password_hash,created_at,updated_at FROM users WHERE email = ?"
	row := a.db.QueryRow(query, email)
	err := row.Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return domain.User{}, err
	}
	return user, nil
}

func (a *AuthRepo) CreateRefreshToken(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error {
	query := "INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)"
	_, err := a.db.Exec(query, userID, tokenHash, expiresAt)
	return err
}

func (a *AuthRepo) GetRefreshToken(ctx context.Context, tokenHash string) (domain.RefreshToken, error) {
	var token domain.RefreshToken
	query := "SELECT id, user_id, token_hash, expires_at,revoked_at,created_at,updated_at,replaced_by_token_id FROM refresh_tokens WHERE token_hash = ?"
	row := a.db.QueryRow(query, tokenHash)
	err := row.Scan(&token.ID, &token.UserID, &token.TokenHash, &token.ExpiresAt, &token.RevokedAt, &token.CreatedAt, &token.UpdatedAt, &token.ReplacedByTokenID)
	if err != nil {
		return domain.RefreshToken{}, err
	}
	return token, nil
}

func (a *AuthRepo) CreateAndUpdateRefreshToken(ctx context.Context, oldTokenID int64, tokenHash string, userID int64, expiresAt time.Time) error {
	tx, err := a.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	var newTokenID int64
	query1 := "INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3) returning id"
	err = tx.QueryRow(query1, userID, tokenHash, expiresAt).Scan(&newTokenID)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			return err
		}
		return err
	}

	query2 := `UPDATE refresh_tokens
				SET revoked_at = NOW(),
					updated_at = NOW(),
					replaced_by_token_id = $1
						WHERE id = $2
				`
	_, err = tx.Exec(query2, newTokenID, oldTokenID)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			return err
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}
