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

	// "github.com/uddinArsalan/ferry/types"
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
	pm.AddGroup("A", "group-A")
	pm.AddGroup("B", "group-B")
	privKey, err := utils.GetPrivKey()
	if err != nil {
		t.Fatal(err)
	}
	eventChan := make(chan types.Event)
	node, err := tcp.NewNode(context.Background(), privKey, pm, eventChan, 10)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(node.GetAddrsAndPort())
	// peersToConnect := []types.PeerLocal{
	// 	{
	// 		Peer: types.Peer{
	// 			ID:      "QmVxts1MuyYrh2VnwNBRbkg4pjzo3UmxeikY7fPK8NzW43",
	// 			Address: "127.0.0.1",
	// 			Name:    "Peer A",
	// 			Port:    51611,
	// 		},
	// 		DirPath: "/A",
	// 	},
	// 	// {
	// 	// 	Peer: types.Peer{
	// 	// 		ID:      "QmPTU5RifvrBUiwvPQxHvFuW4mEcX1sTn47BU7DTDeYgYU",
	// 	// 		Address: "127.0.0.1",
	// 	// 		Name:    "Peer B",
	// 	// 		Port:    51605,
	// 	// 	},
	// 	// 	DirPath: "/B",
	// 	// },
	// }
	// for _, peer := range peersToConnect {
	// 	if err := pm.AddPeerToGroup(types.Peer{
	// 		ID:      peer.Peer.ID,
	// 		Name:    peer.Peer.Name,
	// 		Address: peer.Peer.Address,
	// 		Port:    peer.Peer.Port,
	// 	}, "A", "/a"); err != nil {
	// 		t.Logf("error adding peer to group %v", err)
	// 		continue
	// 	}
	// }
	// node.ConnectToPeers(peersToConnect)
	// node.WriteData("A", tcp.Message{
	// 	Content: "Hello group A members",
	// })
}

func TestNodeInit(t *testing.T) {
	privKeyA, _, err := crypto.GenerateECDSAKeyPair(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pm := peers.NewPeerManager()
	eventChan := make(chan types.Event)
	node, err := tcp.NewNode(context.Background(), privKeyA, pm, eventChan, 10)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("NODE ADDR %v\n", node.GetAddr())
	t.Logf("PEER ID %v\n", node.GetID())

	for peerID := range node.GetConnectedPeers() {
		t.Logf("Connected peer: %v", peerID)
	}
}
