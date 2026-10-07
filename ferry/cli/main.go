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
	pm          peers.PeerManager
	server      tcp.Server
}

func NewCli(ctx context.Context, l *log.Logger, authClient auth.AuthServiceClient, groupClient group.GroupServiceClient) Cli {
	return Cli{
		ctx:         ctx,
		l:           l,
		authClient:  authClient,
		groupClient: groupClient,
		sc:          bufio.NewScanner(os.Stdin),
	}
}

func (c *Cli) TakeUerParams() {
	flag.Parse()
	switch flag.Arg(0) {
	case "login":
		c.login()
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
	if err != SaveToKeyRing(res) {
		c.l.Fatal("Error authenticating")
	}
	c.l.Println(utils.FormatAuthResults(res))
	c.l.Println("Logged in successfully")
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

	if err != SaveToKeyRing(res) {
		c.l.Fatal("Error authenticating")
	}
	c.l.Println(utils.FormatAuthResults(res))
	c.l.Println("Registered successfully")

}

func (c *Cli) createGroup() {
	authContext, err := AuthContext()
	if err != nil {
		c.l.Printf("Unauthenticated request")
	}
	c.l.Println("enter the group name: ")
	c.sc.Scan()
	name := c.sc.Text()
	c.l.Println("Wait till we create your group...")
	c.groupClient.CreateGroup(c.ctx, &group.CreateGroupRequest{Name: name}, authContext)
	c.l.Println("Group created successfully")
}

func (c *Cli) initPeerAndListening() {
	c.l.Println("Init peer creation and listening")
	peerID, err := utils.NewId()
	if err != nil {
		c.l.Println("some error occurred try after some time")
		return
	}
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
	go c.server.Start()
}
