package token

import "time"

type TokenStore interface {
	GenerateToken(userID int64, expiry time.Duration) (string, error)
	VerifyToken(tokenStr string) (*Claims, error)
	GenerateRefreshToken() (string, error)
	HashToken(tokenStr string) string
}
