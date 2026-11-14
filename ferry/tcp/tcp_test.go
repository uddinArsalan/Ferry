package tcp_test

import (
	// "net"
	"context"
	"crypto/rand"
	"testing"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/uddinArsalan/ferry/peers"
	"github.com/uddinArsalan/ferry/tcp"
	"github.com/uddinArsalan/ferry/types"
	"github.com/uddinArsalan/ferry/utils"
)

// func TestTcp(t *testing.T) {
// 	in, err := net.Interfaces()
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	for _, interfac := range in {
// 		addrs, err := interfac.Addrs()
// 		if err != nil {
// 			continue
// 		}
// 		for _, addr := range addrs {
// 			t.Log(addr.String())
// 		}
// 	}

// }

func TestNode(t *testing.T) {
	pm := peers.NewPeerManager()
	privKey, err := utils.GetPrivKey()
	if err != nil {
		t.Fatal(err)
	}

	node, err := tcp.NewNode(context.Background(), privKey, pm)
	if err != nil {
		t.Fatal(err)
	}
	peersToConnect := []types.PeerLocal{
		{
			Peer: types.Peer{
				ID:      "QmYusyMCeeGCBzrRAsiiAbeBecjAQ4N7yKvm9Y6HKntnaD",
				Address: "127.0.0.1",
				Name:    "Peer A",
				Port:    63502,
			},
			DirPath: "/A",
		},
		{
			Peer: types.Peer{
				ID:      "QmRn9vxm1gqBkmBKpUfeBDfoFJqoU5Z973Y4gHrXThRZSQ",
				Address: "127.0.0.1",
				Name:    "Peer B",
				Port:    63611,
			},
			DirPath: "/B",
		},
	}
	node.ConnectToPeers(peersToConnect)
	go node.WriteData()
	select {}
}

func TestNodeInit(t *testing.T) {
	privKeyA, _, err := crypto.GenerateECDSAKeyPair(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pm := peers.NewPeerManager()
	node, err := tcp.NewNode(context.Background(), privKeyA, pm)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("NODE ADDR %v\n", node.GetAddr())
	t.Logf("PEER ID %v\n", node.GetID())

	for peerID := range node.GetConnectedPeers() {
		t.Logf("Connected peer: %v", peerID)
	}
}
