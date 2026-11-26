package main

import (
	// "flag"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/uddinArsalan/ferry/cli"
	"github.com/uddinArsalan/ferry/client"
	"github.com/uddinArsalan/ferry/logger/ferry_logger"
	"github.com/uddinArsalan/ferry/logger/file_logger"
	"github.com/uddinArsalan/ferry/peers"
	"github.com/uddinArsalan/ferry/tcp"
	"github.com/uddinArsalan/ferry/types"
	"github.com/uddinArsalan/ferry/utils"
	"github.com/uddinArsalan/ferry/watcher"
	mapsync "github.com/uddinArsalan/ferry/worker/map_sync"
)

func main() {
	serverAddr := fmt.Sprintf("127.0.0.1:%d", 5051)
	g, err := client.NewGRPCClient(serverAddr)
	if err != nil {
		log.Fatalf("error initialising grpc client conn %v", err)
	}
	authClient := g.NewAuthClient()
	groupClient := g.NewGroupClient()
	peerClient := g.NewPeerClient()
	pm := peers.NewPeerManager()
	ferryLogger := ferry_logger.NewFerryLogger()

	fileLogger, err := file_logger.NewFileLogger()
	if err != nil {
		log.Fatal(err)
	}

	defer fileLogger.Close()

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	// syncer of local maps and configuration
	mapSyncer := mapsync.NewMapSyncer(ctx, fileLogger.GetLogger(), pm, groupClient, 30)
	mapSyncer.Run()

	// node initialisation
	// initialise priv key
	privKey, err := utils.GetPrivKey()
	if err != nil {
		log.Fatal(err)
	}

	eventChan := make(chan types.Event)

	watcher, err := watcher.NewWatcher(eventChan)
	if err != nil {
		log.Fatal(err)
	}

	// creating new node
	node, err := tcp.NewNode(ctx, privKey, pm, eventChan, 20, fileLogger.GetLogger())

	if err != nil {
		log.Fatal(err)
	}
	node.Start()

	cli := cli.NewCli(ctx, ferryLogger.GetLogger(), authClient, groupClient, peerClient, pm, node, watcher)
	cli.TakeUserParams()

	<-ctx.Done()

	err = node.Close()
	if err != nil {
		log.Fatal(err)
	}
}
