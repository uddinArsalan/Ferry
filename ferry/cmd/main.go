package main

import (
	// "flag"
	"context"
	"log"
	"time"

	"github.com/uddinArsalan/ferry/cli"
	"github.com/uddinArsalan/ferry/client"
	// "github.com/uddinArsalan/ferry/peers"
	// "github.com/uddinArsalan/ferry/tcp"
	// "github.com/uddinArsalan/ferry/types"
	// "github.com/uddinArsalan/ferry/watcher"
)

func main() {
	g, err := client.NewDialWithToken()
	if err != nil {
		log.Fatalf("error initialising dial config")
	}
	authClient, err := g.NewAuthClient()
	if err != nil {
		log.Fatalf("error initialising auth client")
	}
	groupClient, err := g.NewGrouplient()
	if err != nil {
		log.Fatalf("error initialising group client")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	logger := cli.NewLogger()
	cli := cli.NewCli(ctx, logger.GetLogger(), authClient, groupClient)
	cli.TakeUerParams()
	// below code will update
	// pm := peers.NewPeerManager()

	// var dir string
	// // var grpId string
	// flag.StringVar(&dir, "dir", "", "directory to watch")
	// // flag.StringVar(&grpId,"grpId","","sync group to add this peer")
	// flag.Parse()

	// w, err := watcher.NewWatcher()
	// if err != nil {
	// 	log.Printf("Error in creating watcher")
	// 	return
	// }

	// eventChan := make(chan types.Event, 10)

	// w.WatchFile(dir, eventChan)

	// server := tcp.NewServer(":3000", pm)

	// go server.Start()

	// go server.ConnectToPeers()

	// go pm.SendToAllPeersOfGroup(eventChan)

	// <-make(chan struct{})
}
