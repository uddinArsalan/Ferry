package client

import (
	"crypto/tls"
	"fmt"
	"log"

	auth "github.com/uddinArsalan/ferry-proto/auth"
	group "github.com/uddinArsalan/ferry-proto/group"
	"github.com/uddinArsalan/ferry/cli"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/credentials/oauth"
)

type gRPCClient struct {
	serverAddr string
}

func NewDialWithToken() (*gRPCClient, error) {
	return &gRPCClient{
		serverAddr: fmt.Sprintf("localhost:%d", 5051),
	}, nil
}

func (g gRPCClient) NewAuthClient() (auth.AuthServiceClient, error) {
	dial := grpc.WithTransportCredentials(insecure.NewCredentials())
	conn, err := grpc.NewClient(g.serverAddr, dial)
	if err != nil {
		log.Printf("err %v", err.Error())
		return nil, err
	}
	return auth.NewAuthServiceClient(conn), nil
}

func (g *gRPCClient) NewGrouplient() (group.GroupServiceClient, error) {
	tokens, err := cli.GetTokens()
	if err != nil {
		return nil, err
	}
	// will add tls support 
	creds := credentials.NewTLS(&tls.Config{
		Certificates: nil,
	})
	tokenSrc := oauth.TokenSource{TokenSource: oauth2.StaticTokenSource(tokens)}
	// Note, the OAuth2 implementation of grpc.PerRPCCredentials requires a
	// client to use grpc.WithTransportCredentials to prevent any insecure transmission of tokens.
	conn, err := grpc.NewClient(
		g.serverAddr,
		// oauth.TokenSource requires the configuration of transport
		// credentials.
		grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(tokenSrc),
	)
	if err != nil {
		return nil, err
	}
	return group.NewGroupServiceClient(conn), nil
}
