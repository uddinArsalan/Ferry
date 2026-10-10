package cli

import (
	"github.com/uddinArsalan/ferry/keyring"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/oauth"
)

// TokenSrc implement TokenSrc
type TokenSrc struct {
}

func (t TokenSrc) Token() (*oauth2.Token, error) {
	tokens, err := keyring.GetTokens()
	if err != nil {
		return nil, err
	}
	return tokens, nil
}

func AuthContext() (grpc.CallOption, error) {
	tokenSrc := oauth.TokenSource{
		TokenSource: TokenSrc{},
	}
	return grpc.PerRPCCredentials(tokenSrc), nil
}
