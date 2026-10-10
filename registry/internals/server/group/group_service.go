package group_service

import (
	"context"
	"log"

	gengroup "github.com/uddinArsalan/ferry-proto/group"
	genpeer "github.com/uddinArsalan/ferry-proto/peer"
	"github.com/uddinArsalan/ferry-registry/interceptor"
	"github.com/uddinArsalan/ferry-registry/internals/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var ErrGroupNotFound = status.Error(codes.NotFound, "Group not found")

type GroupServer struct {
	groupRepo *repository.GroupRepository
	gengroup.UnimplementedGroupServiceServer
}

func NewGroupServer(groupRepo *repository.GroupRepository) *GroupServer {
	return &GroupServer{
		groupRepo: groupRepo,
	}
}

func (g *GroupServer) CreateGroup(ctx context.Context, req *gengroup.CreateGroupRequest) (*gengroup.GroupID, error) {
	groupID, err := g.groupRepo.CreateGroup(ctx, req.GroupId, req.Name)
	if err != nil {
		return nil, err
	}
	return &gengroup.GroupID{
		Id: groupID,
	}, nil
}

func (g *GroupServer) GetGroupsForPeer(ctx context.Context, req *genpeer.GetPeerRequest) (*gengroup.Groups, error) {
	groups, err := g.groupRepo.GetGroupsForPeer(ctx, req.PeerId)
	if err != nil {
		log.Printf("Error retrieving groups for peer %d: %v", req.PeerId, err)
		return nil, err
	}
	var groupList []*gengroup.Group
	for _, group := range groups {
		groupList = append(groupList, &gengroup.Group{
			Id:        group.ID,
			GroupId:   group.GroupID,
			Name:      group.Name,
			UserId:    group.UserID,
			CreatedAt: timestamppb.New(group.CreatedAt),
		})
	}
	return &gengroup.Groups{
		Groups: groupList,
	}, nil
}

func (g *GroupServer) GetGroups(ctx context.Context, void *emptypb.Empty) (*gengroup.Groups, error) {
	userID, ok := interceptor.GetUserIDFromContext(ctx)
	if !ok {
		return nil, interceptor.ErrUnauthenticated
	}
	groups, err := g.groupRepo.GetGroupsForUser(ctx, userID)
	if err != nil {
		log.Printf("Error retrieving groups for user %d: %v", userID, err)
		return nil, err
	}
	var groupList []*gengroup.Group
	for _, group := range groups {
		groupList = append(groupList, &gengroup.Group{
			Id:        group.ID,
			GroupId:   group.GroupID,
			Name:      group.Name,
			UserId:    group.UserID,
			CreatedAt: timestamppb.New(group.CreatedAt),
		})
	}
	return &gengroup.Groups{
		Groups: groupList,
	}, nil
}

func (g *GroupServer) GetGroup(ctx context.Context, req *gengroup.GetGroupRequest) (*gengroup.Group, error) {
	group, err := g.groupRepo.GetGroupByGrpID(ctx, req.GroupId)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, status.Error(codes.NotFound, "Group not found")
	}
	return &gengroup.Group{
		Id:        group.ID,
		GroupId:   group.GroupID,
		Name:      group.Name,
		UserId:    group.UserID,
		CreatedAt: timestamppb.New(group.CreatedAt),
	}, nil
}
