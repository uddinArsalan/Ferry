package main

import (
	"fmt"
	"log"
	"net"

	"github.com/uddinArsalan/ferry-registry/db"
	genauth "github.com/uddinArsalan/ferry-registry/proto/auth"
	gengroup "github.com/uddinArsalan/ferry-registry/proto/group"
	"github.com/uddinArsalan/ferry-registry/repository"
	auth "github.com/uddinArsalan/ferry-registry/server/auth"
	group "github.com/uddinArsalan/ferry-registry/server/group"
	"google.golang.org/grpc"
)

func main(){
	db,err := db.NewDB()
	if err != nil{
		log.Fatalf("Error initialising db connection %v",err.Error())
	}

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", 5051))
	if err != nil{
		log.Fatalf("Error starting grpc server %v",err.Error())
	}

	grpcServer := grpc.NewServer()

	authRepo := repository.NewAuthRepo(db)

	groupServer := group.NewGroupServer()
	authServer := auth.NewAuthService(authRepo)

	gengroup.RegisterGroupServiceServer(grpcServer,groupServer)

	genauth.RegisterAuthServiceServer(grpcServer,authServer)
	
	if err := grpcServer.Serve(lis); err != nil {
        log.Fatal(err)
    }
}