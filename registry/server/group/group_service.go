package group

import (
	"context"

	ferry "github.com/uddinArsalan/ferry-registry/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

type GroupServer struct{
	ferry.UnimplementedGroupServiceServer
}

func NewGroupServer() *GroupServer{
	return &GroupServer{}
}

func (g *GroupServer) RegisterPeer(ctx context.Context,peer *ferry.Peer) (*ferry.Peer, error){
	
}

func (g *GroupServer) CreateGroup(ctx context.Context,void *emptypb.Empty) (*ferry.Group, error){

}

func (g *GroupServer) GetGroupsForPeer(ctx context.Context,perryID *ferry.PeerID) (*ferry.Groups, error){

}

func (g *GroupServer) GetGroups(ctx context.Context,void *emptypb.Empty) (*ferry.Groups, error){

}

func (g *GroupServer) GetPeers(context.Context, *emptypb.Empty) (*ferry.Peers, error){

}

func (g *GroupServer) GetPeer(context.Context, *ferry.PeerID) (*ferry.Peer, error){

}

func (g *GroupServer) GetGroup(context.Context, *ferry.GroupID) (*ferry.Group, error){

}