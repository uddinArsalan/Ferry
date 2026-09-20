package client

import (
	auth "github.com/uddinArsalan/ferry-proto/auth"
	group "github.com/uddinArsalan/ferry-proto/group"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type GRPCClient struct {
	conn *grpc.ClientConn
}

func NewGRPCClient(serverAddr string) (*GRPCClient, error) {
	creds, err := credentials.NewClientTLSFromFile("../certs/myCA.pem", "ferry")
	if err != nil {
		return nil, err
	}
	conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, err
	}
	return &GRPCClient{
		conn: conn,
	}, nil
}

func (g GRPCClient) NewAuthClient() auth.AuthServiceClient {
	return auth.NewAuthServiceClient(g.conn)
}

func (g *GRPCClient) NewGrouplient() group.GroupServiceClient {
	return group.NewGroupServiceClient(g.conn)
}
