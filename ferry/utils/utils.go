package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/uddinArsalan/ferry/keyring"
)

func NewId() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func GetPrivKey() (crypto.PrivKey, error) {
	privKey, err := keyring.GetPrivKey()
	if err == nil && privKey != "" {
		return crypto.UnmarshalECDSAPrivateKey([]byte(privKey))
	}
	cryptoPrivKey, _, err := crypto.GenerateECDSAKeyPair(rand.Reader)
	if err != nil {
		return nil, err
	}
	privKreyBytes, err := cryptoPrivKey.Raw()
	if err != nil {
		return nil, err
	}
	err = keyring.SavePrivKey(string(privKreyBytes))
	if err != nil {
		return nil, err
	}
	return cryptoPrivKey, nil
}

func GetPeerId() (string, error) {
	privKey, err := GetPrivKey()
	if err != nil {
		return "", err
	}
	peerID, err := peer.IDFromPrivateKey(privKey)
	if err != nil {
		return "", err
	}
	return peerID.String(), nil
}

func getDateAndTime(milliseconds int64) time.Time {
	return time.Now().Add(time.Duration(milliseconds) * time.Millisecond)
}

func FormatAuthResults[T keyring.AuthResponse](res T) string {
	return fmt.Sprintf(`
			ACCESS TOKEN : %v\n, REFRESH_TOKEN : %v\n,
			ACCESS_TOKEN_EXPIRES_AT : %v\n,
			REFRESH_TOKEN_EXPIRES_AT :%v
	`, res.GetAccessToken(), res.GetRefreshToken(),
		getDateAndTime(res.GetAccessExpiresAt()),
		getDateAndTime(res.GetRefreshExpiresAt()),
	)
}
