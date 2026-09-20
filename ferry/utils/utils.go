package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

func NewId() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

type AuthResponse interface {
	GetAccessToken() string
	GetRefreshToken() string
	GetAccessExpiresAt() int64
	GetRefreshExpiresAt() int64
}

func getDateAndTime(milliseconds int64) time.Time {
	return time.Now().Add(time.Duration(milliseconds) * time.Millisecond)
}

func FormatAuthResults[T AuthResponse](res T) string {
	return fmt.Sprintf(`
			ACCESS TOKEN : %v\n, REFRESH_TOKEN : %v\n,
			ACCESS_TOKEN_EXPIRES_AT : %v\n,
			REFRESH_TOKEN_EXPIRES_AT :%v
	`, res.GetAccessToken(), res.GetRefreshToken(),
		getDateAndTime(res.GetAccessExpiresAt()),
		getDateAndTime(res.GetRefreshExpiresAt()),
	)
}
