package tests

import (
	"testing"

	"github.com/uddinArsalan/ferry/keyring"
)

func TestToken(t *testing.T) {
	token, err := keyring.GetTokens()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("token %+v", token.RefreshToken)
}
