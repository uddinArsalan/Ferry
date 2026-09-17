package group_service

import (
	"context"

	gengroup "github.com/uddinArsalan/ferry-proto/group"
	"google.golang.org/protobuf/types/known/emptypb"
)

type GroupServer struct {
	gengroup.UnimplementedGroupServiceServer
}

func NewGroupServer() *GroupServer {
	return &GroupServer{}
}

func (g *GroupServer) RegisterPeer(ctx context.Context, peer *gengroup.Peer) (*gengroup.Peer, error) {
	return nil, nil
}

func (g *GroupServer) CreateGroup(ctx context.Context, void *emptypb.Empty) (*gengroup.Group, error) {
	return nil, nil
}

func (g *GroupServer) GetGroupsForPeer(ctx context.Context, perryID *gengroup.PeerID) (*gengroup.Groups, error) {
	return nil, nil
}

func (g *GroupServer) GetGroups(ctx context.Context, void *emptypb.Empty) (*gengroup.Groups, error) {
	return nil, nil
}

func (g *GroupServer) GetPeers(context.Context, *emptypb.Empty) (*gengroup.Peers, error) {
	return nil, nil
}

func (g *GroupServer) GetPeer(context.Context, *gengroup.PeerID) (*gengroup.Peer, error) {
	return nil, nil
}

func (g *GroupServer) GetGroup(context.Context, *gengroup.GroupID) (*gengroup.Group, error) {
	return nil, nil
}
