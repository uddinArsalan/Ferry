package main

import (
	"flag"
	"log"

	"github.com/uddinArsalan/ferry/peers"
	"github.com/uddinArsalan/ferry/tcp"
	"github.com/uddinArsalan/ferry/types"
	"github.com/uddinArsalan/ferry/watcher"
)

func main() {
	pm := peers.NewPeerManager()

	var dir string
	// var grpId string
	flag.StringVar(&dir,"dir","","directory to watch")
	// flag.StringVar(&grpId,"grpId","","sync group to add this peer")
	flag.Parse()

	w, err := watcher.NewWatcher()
	if err != nil {
		log.Printf("Error in creating watcher")
		return
	}

	eventChan := make(chan types.Event,10)

	w.WatchFile(dir,eventChan)

	server := tcp.NewServer(":3000",pm)

	go server.Start()

	go server.ConnectToPeers()

	go pm.SendToAllPeersOfGroup(eventChan) 

	<-make(chan struct{})
}
