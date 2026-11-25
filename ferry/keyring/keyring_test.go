package keyring

import "testing"

func TestGetKeyRing(t *testing.T) {
	tokens, err := GetTokens()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", tokens)
}
