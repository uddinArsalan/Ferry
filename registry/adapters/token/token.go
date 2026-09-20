package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Token struct {
	secret      string
	signingAlgo jwt.SigningMethod
}

type Claims struct {
	jwt.RegisteredClaims
}

func NewToken() Token {
	return Token{
		secret:      os.Getenv("JWT_SECRET"),
		signingAlgo: jwt.SigningMethodHS256,
	}
}

func (j Token) GenerateToken(userID int64, expiry time.Duration) (string, error) {
	now := time.Now()

	token := jwt.NewWithClaims(j.signingAlgo, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(userID, 10),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	})

	return token.SignedString([]byte(j.secret))
}

func (j Token) VerifyToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenStr,
		&Claims{},
		func(t *jwt.Token) (any, error) {
			if t.Method != j.signingAlgo {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(j.secret), nil
		},
	)

	if err != nil{
		return nil, err
	}

	if !token.Valid{
		return nil,errors.New("unauthenticated or invalid tokens")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("invalid claims type")
	}

	return claims, nil
}

func (j Token) GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func(j Token) HashToken(tokenStr string) string {
	hash := sha256.Sum256([]byte(tokenStr))
	return hex.EncodeToString(hash[:])
}
