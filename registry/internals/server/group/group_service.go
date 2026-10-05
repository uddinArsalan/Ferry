package group_service

import (
	"context"
	"log"

	gengroup "github.com/uddinArsalan/ferry-proto/group"
	genpeer "github.com/uddinArsalan/ferry-proto/peer"
	"github.com/uddinArsalan/ferry-registry/interceptor"
	"github.com/uddinArsalan/ferry-registry/internals/repository"
	"google.golang.org/protobuf/types/known/emptypb"
)

type GroupServer struct {
	groupRepo *repository.GroupRepository
	gengroup.UnimplementedGroupServiceServer
}

func NewGroupServer(groupRepo *repository.GroupRepository) *GroupServer {
	return &GroupServer{
		groupRepo: groupRepo,
	}
}

func (g *GroupServer) CreateGroup(ctx context.Context, req *gengroup.CreateGroupRequest) (*gengroup.Group, error) {
	groupID, err := g.groupRepo.CreateGroup(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	return &gengroup.GroupID{
		Id: groupID,
	}, nil
}

func (g *GroupServer) GetGroupsForPeer(ctx context.Context, peerID *genpeer.PeerID) (*gengroup.Groups, error) {
	return nil, nil
}

func (g *GroupServer) GetGroups(ctx context.Context, void *emptypb.Empty) (*gengroup.Groups, error) {
	userID, ok := interceptor.GetUserIDFromContext(ctx)
	if !ok {
		return nil, interceptor.ErrUnauthenticated
	}
	return nil, nil
}

func (g *GroupServer) GetGroup(context.Context, *gengroup.GroupID) (*gengroup.Group, error) {
	return nil, nil
}
