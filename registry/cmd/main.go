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
	"github.com/uddinArsalan/ferry-registry/interceptor"
	"github.com/uddinArsalan/ferry-registry/internals/db"
	"github.com/uddinArsalan/ferry-registry/internals/repository"
	auth "github.com/uddinArsalan/ferry-registry/internals/server/auth"
	group "github.com/uddinArsalan/ferry-registry/internals/server/group"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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
	creds, err := credentials.NewServerTLSFromFile("../certs/ferry.crt", "../certs/ferry.key")
	if err != nil {
		log.Fatalf("failed to create credentials: %v", err)
	}

	authInterceptor := interceptor.NewAuthInterceptor(tokenStore)

	grpcServer := grpc.NewServer(grpc.Creds(creds), grpc.UnaryInterceptor(authInterceptor.UnaryInterceptor))

	authRepo := repository.NewAuthRepo(db)

	groupServer := group.NewGroupServer()
	authServer := auth.NewAuthService(authRepo, passwordStore, tokenStore)

	gengroup.RegisterGroupServiceServer(grpcServer, groupServer)

	genauth.RegisterAuthServiceServer(grpcServer, authServer)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
