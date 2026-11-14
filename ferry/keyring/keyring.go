package keyring

import (
	"strconv"

	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

var (
	service = "ferry"
)

type AuthResponse interface {
	GetAccessToken() string
	GetRefreshToken() string
	GetAccessExpiresAt() int64
	GetRefreshExpiresAt() int64
}

func SaveToKeyRing(authRes AuthResponse) error {
	accessTokenExpiry := strconv.FormatInt(authRes.GetAccessExpiresAt(), 10)
	refreshTokenExpiry := strconv.FormatInt(authRes.GetRefreshExpiresAt(), 10)
	if err := keyring.Set(service, "access_token", authRes.GetAccessToken()); err != nil {
		return err
	}
	if err := keyring.Set(service, "refresh_token", authRes.GetRefreshToken()); err != nil {
		return err
	}
	if err := keyring.Set(service, "access_token_expiry", accessTokenExpiry); err != nil {
		return err
	}
	return keyring.Set(service, "refresh_token_expiry", refreshTokenExpiry)
}

func SavePrivKey(privKey string) error {
	return keyring.Set(service, "priv_key", privKey)
}

func GetPrivKey() (string, error) {
	return keyring.Get(service, "priv_key")
}

func GetTokens() (*oauth2.Token, error) {
	accessToken, err := keyring.Get(service, "access_token")
	if err != nil {
		return nil, err
	}
	refreshToken, err := keyring.Get(service, "refresh_token")
	if err != nil {
		return nil, err
	}
	return &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
	}, nil
}
