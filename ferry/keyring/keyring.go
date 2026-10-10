package keyring

import (
	"strconv"

	"github.com/uddinArsalan/ferry/utils"
	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

var (
	service = "ferry"
)

func SaveToKeyRing(authRes utils.AuthResponse) error {
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

func SavePeerID(localPeerId string) error {
	return keyring.Set(service, "peer_id", localPeerId)
}

func GetPeerId() (string, error) {
	return keyring.Get(service, "peer_id")
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
