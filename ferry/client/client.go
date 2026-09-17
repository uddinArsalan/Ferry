package client

import (
	"fmt"
	"log"

	auth "github.com/uddinArsalan/ferry-proto/auth"
	group "github.com/uddinArsalan/ferry-proto/group"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewAuthClient() (auth.AuthServiceClient, error) {
	serverAddr := fmt.Sprintf("localhost:%d", 5051)
	dial := grpc.WithTransportCredentials(insecure.NewCredentials())
	conn, err := grpc.NewClient(serverAddr, dial)
	if err != nil {
		log.Printf("err %v", err.Error())
		return nil, err
	}
	return auth.NewAuthServiceClient(conn), nil
}

func NewGrouplient() (group.GroupServiceClient, error) {
	serverAddr := fmt.Sprintf("localhost:%d", 5051)
	dial := grpc.WithTransportCredentials(insecure.NewCredentials())
	conn, err := grpc.NewClient(serverAddr, dial)
	if err != nil {
		return nil, err
	}
	return group.NewGroupServiceClient(conn), nil
}
