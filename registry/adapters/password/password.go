package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

type argonParams struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLen     uint32
	KeyLen      uint32
}

type parsedPasswd struct {
	Version int
	Salt    []byte
	Key     []byte
	params  argonParams
}

type Password struct{}

func NewPasswordManager() Password {
	return Password{}
}

func (p Password) HashAndEncodePassword(password string) (string, error) {
	params := argonParams{
		Memory:      20 * 1024,
		Iterations:  2,
		Parallelism: 1,
		KeyLen:      32,
		SaltLen:     16,
	}
	salt, err := p.generateRandomSalt(params.SaltLen)
	if err != nil {
		return "", err
	}
	key := p.hashPassword(password, salt, params)
	// $argon2id$v=19$m=65536,t=3,p=2$c29tZXNhbHQ$RdescudvJCsgt3ub+b+dWRWJTmaaJObG
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Key := base64.RawStdEncoding.EncodeToString(key)

	hash := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, params.Memory, params.Iterations, params.Parallelism, b64Salt, b64Key)
	return hash, nil
}

func (p Password) ComparePasswordAndHash(userPassword string, hash string) (bool, error) {
	return p.checkPassword(userPassword, hash)
}

func (p Password) generateRandomSalt(saltLen uint32) ([]byte, error) {
	b := make([]byte, saltLen)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (p Password) hashPassword(password string, salt []byte, params argonParams) []byte {
	return argon2.IDKey([]byte(password), salt, params.Iterations, params.Memory, params.Parallelism, params.KeyLen)
}

var ErrInvalidHash = errors.New("invalid hash")

func (p Password) decodePassword(passwordHash string) (parsedPasswd, error) {
	parts := strings.Split(passwordHash, "$")
	if len(parts) != 6 {
		return parsedPasswd{}, ErrInvalidHash
	}

	if parts[1] != "argon2id" {
		return parsedPasswd{}, ErrInvalidHash
	}

	var version int
	_, err := fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return parsedPasswd{}, ErrInvalidHash
	}

	if version != argon2.Version {
		return parsedPasswd{}, ErrInvalidHash
	}

	config := strings.Split(parts[3], ",")
	if len(config) != 3 {
		return parsedPasswd{}, ErrInvalidHash
	}

	var (
		memory      uint32
		iterations  uint32
		parallelism uint8
	)
	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism)
	if err != nil {
		return parsedPasswd{}, ErrInvalidHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return parsedPasswd{}, err
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return parsedPasswd{}, err
	}
	return parsedPasswd{
		Version: version,
		Salt:    salt,
		Key:     key,
		params: argonParams{
			Memory:      memory,
			Iterations:  iterations,
			Parallelism: parallelism,
			SaltLen:     uint32(len(salt)),
			KeyLen:      uint32(len(key)),
		},
	}, nil
}

func (p Password) checkPassword(userPassword string, hash string) (bool, error) {
	// password stored in db
	parsedPasswd, err := p.decodePassword(hash)
	if err != nil {
		return false, err
	}
	//user provide password - userPassword
	userKey := p.hashPassword(userPassword, parsedPasswd.Salt, parsedPasswd.params)
	userKeyLen := len(userKey)
	hashKeyLen := len(parsedPasswd.Key)
	if subtle.ConstantTimeEq(int32(userKeyLen), int32(hashKeyLen)) == 0 {
		return false, nil
	}
	if subtle.ConstantTimeCompare(userKey, parsedPasswd.Key) == 1 {
		return true, nil
	}
	return false, nil
}
