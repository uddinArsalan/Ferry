package domain

import "time"

type RefreshToken struct {
	ID                int64
	TokenHash         string
	UserID            int64
	RevokedAt         *time.Time
	CreatedAt         time.Time
	ExpiresAt         time.Time
	ReplacedByTokenID *int64
	UpdatedAt         time.Time
}
