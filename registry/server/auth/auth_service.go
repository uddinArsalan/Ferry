package auth_service

import (
	"context"

	genauth "github.com/uddinArsalan/ferry-registry/proto/auth"
	"github.com/uddinArsalan/ferry-registry/repository"
	"google.golang.org/protobuf/types/known/emptypb"
)

type AuthService struct {
	authRepo *repository.AuthRepo
	genauth.UnimplementedAuthServiceServer
}

func NewAuthService(authRepo *repository.AuthRepo) *AuthService {
	return &AuthService{
		authRepo: authRepo,
	}
}

func (a *AuthService) Login(ctx context.Context, req *genauth.LoginRequest) (*genauth.LoginResponse, error) {
	return nil, nil
}

func (a *AuthService) Refresh(ctx context.Context, void *emptypb.Empty) (*genauth.LoginResponse, error) {
	return nil, nil
}
