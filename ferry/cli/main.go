package cli

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/uddinArsalan/ferry-proto/auth"
	"github.com/uddinArsalan/ferry-proto/group"
	"github.com/uddinArsalan/ferry-proto/peer"
	"github.com/uddinArsalan/ferry/keyring"
	"github.com/uddinArsalan/ferry/peers"
	"github.com/uddinArsalan/ferry/tcp"
	"github.com/uddinArsalan/ferry/utils"
	"golang.org/x/term"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Cli struct {
	ctx         context.Context
	l           *log.Logger
	authClient  auth.AuthServiceClient
	groupClient group.GroupServiceClient
	peerClient  peer.PeerServiceClient
	sc          *bufio.Scanner
	pm          *peers.PeerManager
	server      *tcp.Server
}

func NewCli(
	ctx context.Context,
	l *log.Logger,
	authClient auth.AuthServiceClient,
	groupClient group.GroupServiceClient,
	peerClient peer.PeerServiceClient,
	pm *peers.PeerManager,
	server *tcp.Server,
) Cli {
	return Cli{
		ctx:         ctx,
		l:           l,
		authClient:  authClient,
		groupClient: groupClient,
		peerClient:  peerClient,
		pm:          pm,
		server:      server,
		sc:          bufio.NewScanner(os.Stdin),
	}
}

func (c *Cli) TakeUerParams() {
	flag.Parse()
	switch flag.Arg(0) {
	case "login":
		c.login()
	case "init":
		c.initPeer()
	case "refresh":
		c.refresh()
	case "create":
		switch flag.Arg(1) {
		case "group":
			c.createGroup()
		}
	case "--help":
		// will need to update
		c.l.Printf("Use login to get started")
	default:
		c.l.Printf("No params or invalid params provided")
	}
}

func (c *Cli) login() {
	c.l.Println("Enter your email:")
	c.sc.Scan()
	email := c.sc.Text()

	c.l.Println("Enter your password:")
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		c.l.Printf("error reading password, please try again")
		return
	}
	authRequest := auth.LoginRequest{
		Email:    email,
		Password: string(password),
	}
	c.l.Println("Wait some time .....")
	res, err := c.authClient.Login(c.ctx, &authRequest)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			c.l.Println("No user is found with given email")
			c.l.Println("Would you like to register instead ?")
			c.l.Println("Yes(Y) or No(N) ?")
			c.sc.Scan()
			ans := strings.ToLowerSpecial(unicode.CaseRanges, c.sc.Text())
			if err := c.sc.Err(); err != nil {
				c.l.Fatalf("error:%v", err)
			}
			if ans == "yes" || ans == "y" {
				c.registerUser(&authRequest)
			}
			return
		}
		fmt.Println("Error authenticating", err.Error())
		return
	}
	if err = keyring.SaveToKeyRing(res); err != nil {
		c.l.Fatal("Error authenticating")
	}
	c.l.Println("Logged in successfully")
	c.l.Println(utils.FormatAuthResults(res))
}

func (c *Cli) refresh() {
	authContext, err := AuthContext()
	if err != nil {
		c.l.Fatal("Unauthenticated request")
	}
	c.l.Println("Refreshing tokens.")
	token, err := keyring.GetTokens()
	if err != nil {
		c.l.Fatalf("error in getting token %v", err.Error())
	}
	refreshReq := &auth.RefreshRequest{
		RefreshToken: token.RefreshToken,
	}
	res, err := c.authClient.Refresh(c.ctx, refreshReq, authContext)
	if err != nil {
		c.l.Fatal("error in token refresh")
	}
	c.l.Println("Tokens refreshed successfully...")
	c.l.Println(utils.FormatAuthResults(res))
}

func (c *Cli) registerUser(loginReq *auth.LoginRequest) {
	c.l.Println("Enter your name:")
	c.sc.Scan()
	name := c.sc.Text()
	res, err := c.authClient.Register(c.ctx, &auth.RegisterRequest{
		Email:    loginReq.Email,
		Password: loginReq.Password,
		Name:     name,
	})
	if err != nil {
		c.l.Fatal("Error authenticating")
	}

	if err != keyring.SaveToKeyRing(res) {
		c.l.Fatal("Error authenticating")
	}
	c.l.Println(utils.FormatAuthResults(res))
	c.l.Println("Registered successfully")
}

// sync id represent local grp id
func (c *Cli) createGroup() {
	authContext, err := AuthContext()
	if err != nil {
		c.l.Fatal("Unauthenticated request")
	}
	c.l.Println("enter the group name: ")
	c.sc.Scan()
	name := c.sc.Text()
	c.l.Println("Wait till we create your group...")
	syncID, err := utils.NewId()
	if err != nil {
		c.l.Fatal("error creating group, try after some time")
	}
	c.pm.AddGroup(syncID, name)
	_, err = c.groupClient.CreateGroup(c.ctx,
		&group.CreateGroupRequest{GroupId: syncID, Name: name}, authContext)
	if err != nil {
		c.l.Fatalf("error creating group, try after some time %v", err.Error())
	}
	_, err = c.groupClient.CreatePeerMembership(c.ctx, &group.CreatePeerMembershipRequest{
		GroupId: syncID,
		PeerId:  c.pm.GetPeerId(),
		Role:    group.PeerRole_OWNER,
	})
	if err != nil {
		c.l.Fatalf("error creating group, try after some time %v", err.Error())
	}
	c.l.Printf("Group created successfully Sync ID = %v\n", syncID)
}

func (c *Cli) initPeer() {
	c.l.Println("Init peer creation")
	isPeerExists := true
	peerID, err := keyring.GetPeerId()
	if err != nil {
		isPeerExists = false
		peerID, err = utils.NewId()
		if err != nil {
			c.l.Fatal("some error occurred try after some time")
		}
	}
	if !isPeerExists {
		c.l.Println("enter the peer name: ")
		c.sc.Scan()
		name := c.sc.Text()
		ln := c.server.GetAddressAndPort()
		address := ln[0]
		port, err := strconv.ParseUint(ln[1], 10, 32)
		if err != nil {
			c.l.Println("some error occurred try after some time")
			return
		}
		_, err = c.peerClient.RegisterPeer(c.ctx, &peer.RegisterPeerRequest{
			Name:    name,
			Address: address,
			Port:    uint32(port),
			PeerId:  peerID,
		})
		if err != nil {
			c.l.Println("some error occurred try after some time")
			return
		}
		c.pm.AddPeer(peerID, name, address, uint32(port))
		c.l.Printf("peer created successfully Peer ID = %v", peerID)
	} else {
		peer, err := c.peerClient.GetPeer(c.ctx, &peer.GetPeerRequest{
			PeerId: peerID,
		})
		if err != nil {
			c.l.Fatal("peer init failed")
		}
		c.pm.AddPeer(peerID, peer.Name, peer.Address, peer.Port)
		c.l.Printf("Ferry is already initialized. Peer ID = %v", peerID)
	}
}

func (c *Cli) addPeerToGroup() {
	peerID := flag.String("peer_id", "", "Peer ID")
	grpID := flag.String("group_id", "", "Group ID")
	localDir := flag.String("local_dir", "", "Local Dir")

	flag.Parse()

	if *peerID == "" {
		// we can also define in this way that if peer id not defined
		// peer is self assigning itself to some group
		// so then req goes to the owner of group
		// and it will check and add (for it join req)
		*peerID = c.pm.GetPeerId()
		// c.l.Fatal("Error: --peer_id is required")
	}
	if *grpID == "" {
		c.l.Fatal("Error: --group_id is required")
	}
	if *localDir == "" {
		c.l.Fatal("Error: --peer_id is required")
	}
	// need to persist (grpid,local_dir) so if client disconnects and connect again
	peer, err := c.peerClient.GetPeer(c.ctx, &peer.GetPeerRequest{PeerId: *peerID})
	if err != nil {
		c.l.Fatal("peer not found")
	}
	group, err := c.groupClient.GetGroup(c.ctx, &group.GetGroupRequest{GroupId: *grpID})
	if err != nil {
		c.l.Fatal("group not found")
	}
	// check if curr peer is owner or authorized to add other peer in this group
	// then send joining req to other peer (INVITE)
	// if it accepts then from = curr peer id , to = peer

}

func (c *Cli) start() {
	c.initPeer()
	go c.server.Start()
	go c.getGroups()
	c.server.ConnectToPeers()
}

func (c *Cli) getGroups() {
	for {

	}
}
