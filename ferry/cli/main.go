package cli

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/uddinArsalan/ferry-proto/auth"
	"github.com/uddinArsalan/ferry-proto/group"
	"github.com/uddinArsalan/ferry-proto/peer"
	"github.com/uddinArsalan/ferry/config"
	"github.com/uddinArsalan/ferry/keyring"
	"github.com/uddinArsalan/ferry/peers"
	"github.com/uddinArsalan/ferry/tcp"
	"github.com/uddinArsalan/ferry/utils"
	"github.com/uddinArsalan/ferry/watcher"
	"golang.org/x/term"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Cli struct {
	ctx         context.Context
	l           *log.Logger
	errorLogger *log.Logger
	authClient  auth.AuthServiceClient
	groupClient group.GroupServiceClient
	peerClient  peer.PeerServiceClient
	sc          *bufio.Scanner
	pm          *peers.PeerManager
	server      *tcp.Node
	watcher     *watcher.Watcher
	commandArgs []string
}

type Command struct {
	Name        string
	Description string
	Callback    func() error
}

func NewCli(
	ctx context.Context,
	l *log.Logger,
	authClient auth.AuthServiceClient,
	groupClient group.GroupServiceClient,
	peerClient peer.PeerServiceClient,
	pm *peers.PeerManager,
	server *tcp.Node,
	watcher *watcher.Watcher,
) Cli {
	return Cli{
		ctx:         ctx,
		l:           l,
		errorLogger: log.New(os.Stderr, "[CLI ERROR] ", log.LstdFlags),
		authClient:  authClient,
		groupClient: groupClient,
		peerClient:  peerClient,
		pm:          pm,
		server:      server,
		sc:          bufio.NewScanner(os.Stdin),
		watcher:     watcher,
	}
}

func toSlice(words string) []string {
	return strings.Fields(words)
}

func (c *Cli) getAvailableCommands() map[string]Command {
	commandsMap := map[string]Command{
		"exit": {
			Name:        "exit",
			Description: "exit the CLI",
			Callback:    c.exit,
		},
		"help": {
			Name:        "help",
			Description: "show available commands",
			Callback:    c.help,
		},
		"login": {
			Name:        "login",
			Description: "log in",
			Callback:    c.login,
		},
		"init": {
			Name:        "init",
			Description: "initialize this peer",
			Callback:    c.initPeer,
		},
		"start": {
			Name:        "start",
			Description: "start syncing configured groups",
			Callback:    c.start,
		},
		"refresh": {
			Name:        "refresh",
			Description: "refresh login tokens",
			Callback:    c.refresh,
		},
		"invite peer": {
			Name:        "invite peer <peer_id> <group_id>",
			Description: "invite a peer to a group",
			Callback:    func() error { c.invitePeerToGroup(); return nil },
		},
		"invite accept": {
			Name:        "invite accept <request_id>",
			Description: "accept a peer invitation",
			Callback:    func() error { c.acceptInvite(); return nil },
		},
		"invite reject": {
			Name:        "invite reject <request_id>",
			Description: "reject a peer invitation",
			Callback:    func() error { c.rejectInvite(); return nil },
		},
		"join": {
			Name:        "join <group_id>",
			Description: "request to join a group",
			Callback:    func() error { c.sendJoinReqToGroup(); return nil },
		},
		"join accept": {
			Name:        "join accept <request_id>",
			Description: "accept a group join request",
			Callback:    func() error { c.acceptJoin(); return nil },
		},
		"join reject": {
			Name:        "join reject <request_id>",
			Description: "reject a group join request",
			Callback:    func() error { c.rejectJoin(); return nil },
		},
		"get pending invites": {
			Name:        "get pending invites",
			Description: "show pending invitations",
			Callback:    func() error { c.getAllPendingInvites(); return nil },
		},
		"get join-requests sent": {
			Name:        "get join-requests sent",
			Description: "show join requests sent by this peer",
			Callback:    func() error { c.getAllJoinReqByPeer(); return nil },
		},
		"get join-requests received": {
			Name:        "get join-requests received",
			Description: "show join requests received by this peer",
			Callback:    func() error { c.getAllJoinReqForPeer(); return nil },
		},
		"create group": {
			Name:        "create group",
			Description: "create a group",
			Callback:    func() error { c.createGroup(); return nil },
		},
	}
	return commandsMap
}

func (c *Cli) TakeUserParams() {
	for {
		fmt.Print("> ")
		if !c.sc.Scan() {
			if err := c.sc.Err(); err != nil {
				c.reportError("read command", err)
			}
			return
		}
		c.commandArgs = toSlice(c.sc.Text())
		if len(c.commandArgs) == 0 {
			continue
		}

		command, ok := c.findCommand(c.commandArgs)
		if !ok {
			c.l.Println("Unknown command. Type 'help' to see available commands.")
			continue
		}
		if err := command.Callback(); err != nil {
			c.reportError(command.Name, err)
		}
		if strings.EqualFold(c.commandArgs[0], "exit") {
			return
		}
	}
}

func (c *Cli) findCommand(args []string) (Command, bool) {
	commands := c.getAvailableCommands()
	for end := len(args); end > 0; end-- {
		name := strings.ToLower(strings.Join(args[:end], " "))
		if command, ok := commands[name]; ok {
			return command, true
		}
	}
	return Command{}, false
}

func (c *Cli) exit() error {
	return nil
}

func (c *Cli) help() error {
	c.l.Println("Available commands:")
	commands := c.getAvailableCommands()
	names := make([]string, 0, len(commands))
	for name := range commands {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		cmd := commands[name]
		c.l.Printf("  %-36s %s", cmd.Name, cmd.Description)
	}
	return nil
}

func (c *Cli) reportError(action string, err error) {
	c.errorLogger.Printf("%s: %v", action, err)
}

func (c *Cli) readInput(prompt string) (string, error) {
	c.l.Println(prompt)
	if !c.sc.Scan() {
		return "", c.sc.Err()
	}
	return c.sc.Text(), nil
}

func (c *Cli) login() error {
	email, err := c.readInput("Enter your email:")
	if err != nil {
		return fmt.Errorf("read email: %w", err)
	}
	if email == "" {
		c.l.Println("Email is required.")
		return nil
	}

	c.l.Println("Enter your password:")
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("read password: %w", err)
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
			ans, inputErr := c.readInput("Would you like to register instead? Yes (Y) or No (N)")
			if inputErr != nil {
				return fmt.Errorf("read registration choice: %w", inputErr)
			}
			if strings.EqualFold(ans, "yes") || strings.EqualFold(ans, "y") {
				return c.registerUser(&authRequest)
			}
			return nil
		}
		return fmt.Errorf("authenticate: %w", err)
	}
	if err = keyring.SaveToKeyRing(res); err != nil {
		return fmt.Errorf("save authentication tokens: %w", err)
	}
	c.l.Println("Logged in successfully")
	return nil
}

func (c *Cli) refresh() error {
	c.l.Println("Refreshing tokens.")
	token, err := keyring.GetTokens()
	if err != nil {
		return fmt.Errorf("read saved tokens: %w", err)
	}
	refreshReq := &auth.RefreshRequest{
		RefreshToken: token.RefreshToken,
	}
	res, err := c.authClient.Refresh(c.ctx, refreshReq)
	if err != nil {
		return fmt.Errorf("refresh tokens: %w", err)
	}
	if err = keyring.SaveToKeyRing(res); err != nil {
		return fmt.Errorf("save refreshed tokens: %w", err)
	}
	c.l.Println("Tokens refreshed successfully...")
	return nil
}

func (c *Cli) registerUser(loginReq *auth.LoginRequest) error {
	name, err := c.readInput("Enter your name:")
	if err != nil {
		return fmt.Errorf("read name: %w", err)
	}
	if name == "" {
		c.l.Println("Name is required.")
		return nil
	}

	res, err := c.authClient.Register(c.ctx, &auth.RegisterRequest{
		Email:    loginReq.Email,
		Password: loginReq.Password,
		Name:     name,
	})
	if err != nil {
		return fmt.Errorf("register user: %w", err)
	}

	if err = keyring.SaveToKeyRing(res); err != nil {
		return fmt.Errorf("save authentication tokens: %w", err)
	}
	c.l.Println("Registered successfully")
	return nil
}

// when this peer create a group it should also provide its local dir
// whether it will be a part of group or not ?
// if not then it will just invite other peers that will provide local dir and
// this will just be a admin/owner for req and all

// otherwise it will be the owner as well as the member

// sync id represent local grp id
func (c *Cli) createGroup() {
	authContext, err := AuthContext()
	if err != nil {
		c.reportError("create group: get authentication", err)
		return
	}
	name, err := c.readInput("Enter the group name:")
	if err != nil {
		c.reportError("create group: read group name", err)
		return
	}
	if name == "" {
		c.l.Println("Group name is required.")
		return
	}
	c.l.Println("Wait till we create your group...")

	groupID, err := utils.NewId()
	if err != nil {
		c.reportError("create group ID", err)
		return
	}
	_, err = c.groupClient.CreateGroup(c.ctx,
		&group.CreateGroupRequest{GroupId: groupID, Name: name}, authContext)
	if err != nil {
		c.reportError("create group on server", err)
		return
	}

	currPeerID, err := c.pm.GetPeerId()
	if err != nil {
		c.reportError("get local peer ID", err)
		return
	}

	_, err = c.groupClient.CreateGroupPeerMembership(c.ctx, &group.CreateGroupPeerMembershipRequest{
		GroupId: groupID,
		PeerId:  currPeerID,
		Role:    group.PeerRole_OWNER,
	}, authContext)

	if err != nil {
		c.reportError("add local peer to group", err)
		return
	}
	c.pm.AddGroup(groupID, name)

	localDir, err := c.readInput("Enter the local directory to sync:")
	if err != nil {
		c.reportError("read local directory", err)
		return
	}

	if localDir == "" {
		c.l.Println("Local directory is required.")
		return
	}

	// need to persist (groupid,local_dir) so if client disconnects and connect again
	err = config.SaveLocalDir(config.LocalDirConfig{
		GroupID:  groupID,
		LocalDir: localDir,
	})
	if err != nil {
		c.reportError("save local directory config", err)
		return
	}

	if err = c.pm.AddPeerToGroup(currPeerID, groupID, localDir); err != nil {
		c.reportError("add local peer to group", err)
		return
	}
	go c.watcher.WatchFile(localDir, groupID)

	c.l.Printf("Group created successfully Group ID = %v\n", groupID)
}

func (c *Cli) initPeer() error {
	authContext, err := AuthContext()
	if err != nil {
		return fmt.Errorf("get authentication: %w", err)
	}
	c.l.Println("Init peer creation")
	peerID, err := utils.GetPeerId()
	if err != nil {
		return fmt.Errorf("get local peer ID: %w", err)
	}
	existingPeer, err := c.peerClient.GetPeer(c.ctx, &peer.GetPeerRequest{
		PeerId: peerID,
	}, authContext)
	if err != nil {
		if status.Code(err) != codes.NotFound {
			return fmt.Errorf("look up peer: %w", err)
		}
		name, inputErr := c.readInput("Enter the peer name:")
		if inputErr != nil {
			return fmt.Errorf("read peer name: %w", inputErr)
		}
		if name == "" {
			c.l.Println("Peer name is required.")
			return nil
		}
		ln := c.server.GetAddrsAndPort()
		if len(ln) < 2 {
			return fmt.Errorf("get peer address and port: unexpected address %q", ln)
		}
		address := ln[0]
		port, err := strconv.ParseUint(ln[1], 10, 32)
		if err != nil {
			return fmt.Errorf("parse peer port %q: %w", ln[1], err)
		}
		_, err = c.peerClient.RegisterPeer(c.ctx, &peer.RegisterPeerRequest{
			Name:    name,
			Address: address,
			Port:    uint32(port),
			PeerId:  peerID,
		}, authContext)
		if err != nil {
			return fmt.Errorf("register peer: %w", err)
		}
		c.pm.AddPeer(peerID, name, address, uint32(port))
		c.l.Printf("peer created successfully Peer ID = %v", peerID)
	} else {
		if existingPeer == nil {
			return fmt.Errorf("look up peer: server returned an empty peer")
		}
		c.pm.AddPeer(peerID, existingPeer.Name, existingPeer.Address, existingPeer.Port)
		c.l.Printf("Ferry is already initialized. Peer ID = %v", peerID)
	}
	return nil
}

func (c *Cli) invitePeerToGroup() {
	authContext, err := AuthContext()
	if err != nil {
		c.reportError("invite peer: get authentication", err)
		return
	}
	peerID, groupID := c.commandArg(2), c.commandArg(3)

	currPeerId, err := c.pm.GetPeerId()
	if err != nil {
		c.reportError("invite peer: get local peer ID", err)
		return
	}
	if currPeerId == "" {
		c.l.Println("Peer is not initialized. Run 'init' first.")
		return
	}

	if peerID == "" {
		c.l.Println("Peer ID is required. Usage: invite peer <peer_id> <group_id>")
		return
	}
	if groupID == "" {
		c.l.Println("Group ID is required. Usage: invite peer <peer_id> <group_id>")
		return
	}

	_, err = c.peerClient.GetPeer(c.ctx, &peer.GetPeerRequest{PeerId: peerID}, authContext)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			c.l.Println("Peer not found.")
			return
		}
		c.reportError("invite peer: look up recipient peer", err)
		return
	}
	_, err = c.groupClient.GetGroup(c.ctx, &group.GetGroupRequest{GroupId: groupID}, authContext)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			c.l.Println("Group not found.")
			return
		}
		c.reportError("invite peer: look up group", err)
		return
	}
	// we need to check if the peer we want to invite is already a part
	// of the provided group or not
	mem, err := c.groupClient.GetGroupPeerMembership(c.ctx, &group.GetGroupPeerMembershipRequest{
		GroupId: groupID,
		PeerId:  peerID,
	}, authContext)

	if err != nil {
		if status.Code(err) == codes.NotFound {

		} else {
			c.reportError("invite peer: check recipient membership", err)
			return
		}
	} else {
		c.l.Println("peer is already a member of group")
		return
	}
	// check if curr peer is owner or authorized to add other peer in this group
	// then send joining req to other peer (INVITE)
	mem, err = c.groupClient.GetGroupPeerMembership(c.ctx, &group.GetGroupPeerMembershipRequest{
		GroupId: groupID,
		PeerId:  currPeerId,
	}, authContext)
	if err != nil {
		c.reportError("invite peer: check local membership", err)
		return
	}
	if mem.Role == group.PeerRole_MEMBER {
		c.l.Println("the current peer is not authorized to perform this.")
		return
	}
	_, err = c.groupClient.CreateJoinRequest(c.ctx, &group.CreateJoin{
		GroupId:         groupID,
		Type:            group.JoinType_INVITE,
		SenderPeerId:    currPeerId,
		RecipientPeerId: peerID,
		Status:          group.INVITE_STATUS_PENDING,
	}, authContext)
	if err != nil {
		c.reportError("invite peer: create invitation", err)
		return
	}
	c.l.Println("Peer Invited successfully")
}

// will give all pending invites for the current peer (by other peer group owners)
func (c *Cli) getAllPendingInvites() {
	authContext, err := AuthContext()
	if err != nil {
		c.reportError("list invitations: get authentication", err)
		return
	}
	currPeerId, err := c.pm.GetPeerId()
	if err != nil {
		c.reportError("list invitations: get local peer ID", err)
		return
	}
	if currPeerId == "" {
		c.l.Println("Peer is not initialized. Run 'init' first.")
		return
	}
	pendingInvites, err := c.groupClient.GetAllPendingInvites(c.ctx, &group.GetAllPendingRequest{
		PeerId: currPeerId,
	}, authContext)
	if err != nil {
		c.reportError("list invitations", err)
		return
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

// since we are showing only curr peer pending invites
// this peer is authorize to make decisions of each req
// either accept or reject
func (c *Cli) rejectInvite() {
	authContext, err := AuthContext()
	if err != nil {
		c.reportError("reject invitation: get authentication", err)
		return
	}
	requestID, ok := c.requestIDArg(2, "invite reject <request_id>")
	if !ok {
		return
	}

	req, err := c.groupClient.GetJoinRequest(c.ctx, &group.GetJoinReq{
		RequestId: requestID,
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			c.l.Println("Invitation request not found.")
		} else {
			c.reportError("reject invitation: get request", err)
		}
		return
	}
	if req.Type == group.JoinType_JOIN_REQUEST {
		c.l.Println("That request is a join request, not an invitation.")
		return
	}
	_, err = c.groupClient.UpdateJoinRequestStatus(c.ctx, &group.UpdateJoin{
		RequestId: requestID,
		Status:    group.INVITE_STATUS_REJECTED,
	}, authContext)
	if err != nil {
		c.reportError("reject invitation", err)
		return
	}
}

func (c *Cli) acceptInvite() {
	authContext, err := AuthContext()
	if err != nil {
		c.reportError("accept invitation: get authentication", err)
		return
	}
	requestID, ok := c.requestIDArg(2, "invite accept <request_id>")
	if !ok {
		return
	}

	req, err := c.groupClient.GetJoinRequest(c.ctx, &group.GetJoinReq{
		RequestId: requestID,
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			c.l.Println("Invitation request not found.")
		} else {
			c.reportError("accept invitation: get request", err)
		}
		return
	}
	localDir, err := c.readInput("Enter the local directory to sync:")
	if err != nil {
		c.reportError("accept invitation: read local directory", err)
		return
	}

	if localDir == "" {
		c.l.Println("Local directory is required.")
		return
	}
	_, err = c.groupClient.UpdateJoinRequestStatus(c.ctx, &group.UpdateJoin{
		RequestId: requestID,
		Status:    group.INVITE_STATUS_ACCEPTED,
	}, authContext)
	if err != nil {
		c.reportError("accept invitation", err)
		return
	}

	// need to persist (grpid,local_dir) so if client disconnects and connect again
	err = config.SaveLocalDir(config.LocalDirConfig{
		GroupID:  req.GroupId,
		LocalDir: localDir,
	})
	if err != nil {
		c.reportError("accept invitation: save local directory config", err)
		return
	}
	c.pm.AddGroup(req.GroupId, req.GroupName)

	currPeerID, err := c.pm.GetPeerId()
	if err != nil {
		c.reportError("accept invitation: get local peer ID", err)
		return
	}

	if err = c.pm.AddPeerToGroup(currPeerID, req.GroupId, localDir); err != nil {
		c.reportError("accept invitation: add local peer to group", err)
		return
	}
	go c.watcher.WatchFile(localDir, req.GroupId)
}

// will give all pending join req send by a curr peer to peer group owners
// across other groups to join in groups, only those peer owners can accept or reject
func (c *Cli) getAllJoinReqByPeer() {
	authContext, err := AuthContext()
	if err != nil {
		c.reportError("list sent join requests: get authentication", err)
		return
	}
	currPeerID, err := c.pm.GetPeerId()
	if err != nil {
		c.reportError("list sent join requests: get local peer ID", err)
		return
	}
	if currPeerID == "" {
		c.l.Println("Peer is not initialized. Run 'init' first.")
		return
	}
	// need to either do ferry start or run (need to decide)
	pendingJoinReq, err := c.groupClient.GetAllPendingJoinRequestsBySender(c.ctx, &group.GetAllPendingRequest{
		PeerId: currPeerID,
	}, authContext)
	if err != nil {
		c.reportError("list sent join requests", err)
		return
	}
	for _, join := range pendingJoinReq.PendingJoinRequests {
		c.l.Printf(
			`Invite Req : %v\n
			Group ID : %v\n
			Group Name : %v\n
			Sender Peer Name : %v\n
			Recipient Peer Name : %v\n\n
			`,
			join.RequestId,
			join.GroupId,
			join.GroupName,
			join.SenderPeerName,
			join.RecipientPeerName,
		)
	}
}

// will give all pending join req send by a peers for this
// curr peer across all groups to join in a group
func (c *Cli) getAllJoinReqForPeer() {
	authContext, err := AuthContext()
	if err != nil {
		c.reportError("list received join requests: get authentication", err)
		return
	}
	currPeerID, err := c.pm.GetPeerId()
	if err != nil {
		c.reportError("list received join requests: get local peer ID", err)
		return
	}
	if currPeerID == "" {
		c.l.Println("Peer is not initialized. Run 'init' first.")
		return
	}
	pendingJoinReq, err := c.groupClient.GetAllPendingJoinRequestsForOwner(c.ctx, &group.GetAllPendingRequest{
		PeerId: currPeerID,
	}, authContext)
	if err != nil {
		c.reportError("list received join requests", err)
		return
	}
	for _, join := range pendingJoinReq.PendingJoinRequests {
		c.l.Printf(
			`Invite Req : %v\n
			 Group ID : %v\n
			 Group Name : %v\n
			 Sender Peer Name : %v\n
			 Recipient Peer Name : %v\n\n
			`,
			join.RequestId,
			join.GroupId,
			join.GroupName,
			join.SenderPeerName,
			join.RecipientPeerName,
		)
	}
}

func (c *Cli) acceptJoin() {
	authContext, err := AuthContext()
	if err != nil {
		c.reportError("accept join request: get authentication", err)
		return
	}
	requestID, ok := c.requestIDArg(2, "join accept <request_id>")
	if !ok {
		return
	}

	req, err := c.groupClient.GetJoinRequest(c.ctx, &group.GetJoinReq{
		RequestId: requestID,
	})

	if err != nil {
		if status.Code(err) == codes.NotFound {
			c.l.Println("Join request not found.")
		} else {
			c.reportError("accept join request: get request", err)
		}
		return
	}

	if req.Type == group.JoinType_INVITE {
		c.l.Println("That request is an invitation, not a join request.")
		return
	}

	// need to decide whether curr peer is authorized to perform this,
	// for this request id
	localDir, err := c.readInput("Enter the local directory to sync:")
	if err != nil {
		c.reportError("accept join request: read local directory", err)
		return
	}

	if localDir == "" {
		c.l.Println("Local directory is required.")
		return
	}
	_, err = c.groupClient.UpdateJoinRequestStatus(c.ctx, &group.UpdateJoin{
		RequestId: requestID,
		Status:    group.INVITE_STATUS_ACCEPTED,
	}, authContext)

	if err != nil {
		c.reportError("accept join request", err)
		return
	}

	// need to persist (grpid,local_dir) so if client disconnects and connect again
	err = config.SaveLocalDir(config.LocalDirConfig{
		GroupID:  req.GroupId,
		LocalDir: localDir,
	})
	if err != nil {
		c.reportError("accept join request: save local directory config", err)
		return
	}
	c.pm.AddGroup(req.GroupId, req.GroupName)
	currPeerID, err := c.pm.GetPeerId()
	if err != nil {
		c.reportError("accept join request: get local peer ID", err)
		return
	}
	if err = c.pm.AddPeerToGroup(currPeerID, req.GroupId, localDir); err != nil {
		c.reportError("accept join request: add local peer to group", err)
		return
	}
	go c.watcher.WatchFile(localDir, req.GroupId)
}

func (c *Cli) rejectJoin() {
	authContext, err := AuthContext()
	if err != nil {
		c.reportError("reject join request: get authentication", err)
		return
	}
	requestID, ok := c.requestIDArg(2, "join reject <request_id>")
	if !ok {
		return
	}

	req, err := c.groupClient.GetJoinRequest(c.ctx, &group.GetJoinReq{
		RequestId: requestID,
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			c.l.Println("Join request not found.")
		} else {
			c.reportError("reject join request: get request", err)
		}
		return
	}
	if req.Type == group.JoinType_INVITE {
		c.l.Println("That request is an invitation, not a join request.")
		return
	}
	_, err = c.groupClient.UpdateJoinRequestStatus(c.ctx, &group.UpdateJoin{
		RequestId: requestID,
		Status:    group.INVITE_STATUS_REJECTED,
	}, authContext)
	if err != nil {
		c.reportError("reject join request", err)
		return
	}
}

func (c *Cli) sendJoinReqToGroup() {
	authContext, err := AuthContext()
	if err != nil {
		c.reportError("send join request: get authentication", err)
		return
	}
	groupID := c.commandArg(1)

	if groupID == "" {
		c.l.Println("Group ID is required. Usage: join <group_id>")
		return
	}

	currPeerID, err := c.pm.GetPeerId()
	if err != nil {
		c.reportError("send join request: get local peer ID", err)
		return
	}
	if currPeerID == "" {
		c.l.Println("Peer is not initialized. Run 'init' first.")
		return
	}

	_, err = c.groupClient.GetGroup(c.ctx, &group.GetGroupRequest{GroupId: groupID}, authContext)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			c.l.Println("Group not found.")
			return
		}
		c.reportError("send join request: look up group", err)
		return
	}

	// we need to check if curr peer is already a part
	// of the provided group or not
	_, err = c.groupClient.GetGroupPeerMembership(c.ctx, &group.GetGroupPeerMembershipRequest{
		GroupId: groupID,
		PeerId:  currPeerID,
	}, authContext)

	if err == nil {
		c.l.Println("peer is already a member of group")
		return
	}
	if status.Code(err) != codes.NotFound {
		c.reportError("send join request: check group membership", err)
		return
	}

	// this join req will go for group owner peer (so recipient is group owner)
	// and sender is curr peer that wants to join in group

	peerID, err := c.groupClient.GetGroupOwnerPeerID(c.ctx, &group.GetGroupRequest{
		GroupId: groupID,
	}, authContext)

	if err != nil {
		c.reportError("send join request: get group owner", err)
		return
	}

	_, err = c.groupClient.CreateJoinRequest(c.ctx, &group.CreateJoin{
		GroupId:         groupID,
		Type:            group.JoinType_JOIN_REQUEST,
		SenderPeerId:    currPeerID,
		RecipientPeerId: peerID.PeerId,
		Status:          group.INVITE_STATUS_PENDING,
	}, authContext)
	if err != nil {
		c.reportError("send join request: create request", err)
		return
	}
	c.l.Println("join request sent successfully")
}

func (c *Cli) requestIDArg(index int, usage string) (int64, bool) {
	value := c.commandArg(index)
	if value == "" {
		c.l.Printf("request ID is required (usage: %s)", usage)
		return 0, false
	}
	requestID, err := strconv.ParseInt(value, 10, 64)
	if err != nil || requestID < 0 {
		c.l.Printf("invalid request ID %q (usage: %s)", value, usage)
		return 0, false
	}
	return requestID, true
}

func (c *Cli) commandArg(index int) string {
	if index < 0 || index >= len(c.commandArgs) {
		return ""
	}
	return c.commandArgs[index]
}

func (c *Cli) start() error {
	err := c.initPeer()
	if err != nil {
		return fmt.Errorf("initialize peer: %w", err)
	}
	localDirMap, err := config.GetLocalDirs()
	if err != nil {
		return fmt.Errorf("read local directory config: %w", err)
	}
	for groupID, localDir := range localDirMap {
		err = c.pm.UpdateGroupPeerConfig(groupID, localDir)
		if err != nil {
			c.l.Printf("Group %s is not in the sync snapshot yet; watching %s anyway.", groupID, localDir)
		}
		go c.watcher.WatchFile(localDir, groupID)
	}
	return nil
}
