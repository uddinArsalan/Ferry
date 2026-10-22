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
	"github.com/uddinArsalan/ferry/config"
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
	case "start":
		c.start()
	case "refresh":
		c.refresh()
	case "invite":
		switch flag.Arg(1) {
		case "accept":
			c.acceptInvite()
		case "reject":
			c.rejectInvite()
		}
	case "get":
		switch flag.Arg(1) {
		case "pending":
			switch flag.Arg(2) {
			case "invites":
				c.getAllPendingInvites()
			}
		}
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
	_, err = c.groupClient.CreateGroupPeerMembership(c.ctx, &group.CreateGroupPeerMembershipRequest{
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

func (c *Cli) invitePeerToGroup() {
	peerID := flag.String("peer_id", "", "Peer ID")
	groupID := flag.String("group_id", "", "Group ID")

	flag.Parse()

	currPeerId := c.pm.GetPeerId()
	if currPeerId == "" {
		c.l.Fatalf("unauthenticated or some problem with config")
	}

	if *peerID == "" {
		c.l.Fatal("Error: --peer_id is required")
	}
	if *groupID == "" {
		c.l.Fatal("Error: --group_id is required")
	}

	_, err := c.peerClient.GetPeer(c.ctx, &peer.GetPeerRequest{PeerId: *peerID})
	if err != nil {
		c.l.Fatal("peer not found")
	}
	_, err = c.groupClient.GetGroup(c.ctx, &group.GetGroupRequest{GroupId: *groupID})
	if err != nil {
		c.l.Fatal("group not found")
	}
	// we need to check if the peer we want to invite is already a part
	// of the provided group or not
	mem, err := c.groupClient.GetGroupPeerMembership(c.ctx, &group.GetGroupPeerMembershipRequest{
		GroupId: *groupID,
		PeerId:  *peerID,
	})

	if err != nil {
		if status.Code(err) == codes.NotFound {

		} else {
			c.l.Fatal("error , contact the ferry owner")
		}
	} else {
		c.l.Fatal("peer is already a member of group")
	}

	// check if curr peer is owner or authorized to add other peer in this group
	// then send joining req to other peer (INVITE)
	mem, err = c.groupClient.GetGroupPeerMembership(c.ctx, &group.GetGroupPeerMembershipRequest{
		GroupId: *groupID,
		PeerId:  currPeerId,
	})
	if err != nil {
		c.l.Fatal("some error occurred try after some time")
	}
	if mem.Role == group.PeerRole_MEMBER {
		c.l.Fatal("the current peer is not authorized to perform this.")
	}
	_, err = c.groupClient.CreateJoinRequest(c.ctx, &group.CreateJoin{
		GroupId:         *groupID,
		Type:            group.JoinType_INVITE,
		SenderPeerId:    currPeerId,
		RecipientPeerId: *peerID,
		Status:          group.INVITE_STATUS_PENDING,
	})
	if err != nil {
		c.l.Fatal("error creating group membership")
	}
	c.l.Println("Peer Invited successfully")
}

func (c *Cli) getAllPendingInvites() {
	pendingInvites, err := c.groupClient.GetAllPendingInvites(c.ctx, &group.GetAllPendingRequest{
		PeerId: c.pm.GetPeerId(),
	})
	if err != nil {
		c.l.Fatal("error getting pending invites for curr peer")
	}
	for _, invite := range pendingInvites.PendingInviteRequests {
		c.l.Printf(
			`Invite Req : %v\n
			Group Name : %v\n
			Sender Peer Name : %v\n\n
			`,
			invite.RequestId,
			invite.GroupName,
			invite.SenderPeerName,
		)
	}
}

func (c *Cli) rejectInvite() {
	requestID := flag.Int64("request_id", -1, "Request ID")
	flag.Parse()

	if *requestID == -1 {
		c.l.Fatal("Error: --request_id is required")
	}

	_, err := c.groupClient.GetJoinRequest(c.ctx, &group.GetJoinReq{
		RequestId: *requestID,
	})
	if err != nil {
		c.l.Fatal("invalid req id or err in getting the request")
	}
	_, err = c.groupClient.UpdateJoinRequestStatus(c.ctx, &group.UpdateJoin{
		RequestId: *requestID,
		Status:    group.INVITE_STATUS_REJECTED,
	})
	if err != nil {
		c.l.Fatalf("error rejecting invite with request id %v", requestID)
	}
}

func (c *Cli) acceptInvite() {
	requestID := flag.Int64("request_id", -1, "Request ID")
	flag.Parse()

	if *requestID == -1 {
		c.l.Fatal("Error: --request_id is required")
	}

	req, err := c.groupClient.GetJoinRequest(c.ctx, &group.GetJoinReq{
		RequestId: *requestID,
	})
	if err != nil {
		c.l.Fatal("invalid req id or err in getting the request")
	}
	_, err = c.groupClient.UpdateJoinRequestStatus(c.ctx, &group.UpdateJoin{
		RequestId: *requestID,
		Status:    group.INVITE_STATUS_ACCEPTED,
	})
	if err != nil {
		c.l.Fatalf("error accepting invite with request id %v", requestID)
	}

	c.l.Println("Enter your Local dir that you want to sync in this group:")
	c.sc.Scan()
	localDir := c.sc.Text()

	if localDir == "" {
		c.l.Fatal("Error: local_dir is required")
	}

	// need to persist (grpid,local_dir) so if client disconnects and connect again
	err = config.SaveLocalDir(config.LocalDirConfig{
		GroupID:  req.GroupId,
		LocalDir: localDir,
	})
	if err != nil {
		c.l.Fatal("error adding config")
	}
	err = c.pm.UpdateGroupPeerConfig(req.GroupId, localDir)
	if err != nil {
		c.l.Fatal("error updating local config")
	}
}

func (c *Cli) start() {
	c.initPeer()
	go c.server.Start()
	c.server.ConnectToPeers()
	localDirMap, err := config.GetLocalDirs()
	if err != nil {
		c.l.Fatal("error reading config files")
	}
	for groupID, localDir := range localDirMap {
		err = c.pm.UpdateGroupPeerConfig(groupID, localDir)
		if err != nil {
			c.l.Println("error updating local config")
		}
	}
}
