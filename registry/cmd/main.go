package main

import (
	"fmt"
	"log"
	"net"

	"github.com/joho/godotenv"
	genauth "github.com/uddinArsalan/ferry-proto/auth"
	gengroup "github.com/uddinArsalan/ferry-proto/group"
	"github.com/uddinArsalan/ferry-registry/adapters/password"
	"github.com/uddinArsalan/ferry-registry/adapters/token"
	"github.com/uddinArsalan/ferry-registry/db"
	"github.com/uddinArsalan/ferry-registry/repository"
	auth "github.com/uddinArsalan/ferry-registry/server/auth"
	group "github.com/uddinArsalan/ferry-registry/server/group"
	"google.golang.org/grpc"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	db, err := db.NewDB()
	if err != nil {
		log.Fatalf("Error initialising db connection %v", err.Error())
	}

	tokenStore := token.NewToken()
	passwordStore := password.NewPasswordManager()

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", 5051))
	if err != nil {
		log.Fatalf("Error starting grpc server %v", err.Error())
	}

	grpcServer := grpc.NewServer()

	authRepo := repository.NewAuthRepo(db)

	groupServer := group.NewGroupServer()
	authServer := auth.NewAuthService(authRepo, passwordStore, tokenStore)

	gengroup.RegisterGroupServiceServer(grpcServer, groupServer)

	genauth.RegisterAuthServiceServer(grpcServer, authServer)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
