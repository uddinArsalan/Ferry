package main

import (
	"fmt"
	"log"
	"net"

	ferry "github.com/uddinArsalan/ferry-registry/proto"
	"github.com/uddinArsalan/ferry-registry/server/group"
	"google.golang.org/grpc"
)

func main(){
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", 5051))
	if err != nil{
		log.Fatalf("Error starting grpc server %v",err.Error())
	}
	grpcServer := grpc.NewServer()

	groupServer := group.NewGroupServer()

	ferry.RegisterGroupServiceServer(grpcServer,groupServer)
	if err := grpcServer.Serve(lis); err != nil {
        log.Fatal(err)
    }
}