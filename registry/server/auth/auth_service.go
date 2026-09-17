package auth_service

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"

	genauth "github.com/uddinArsalan/ferry-proto/auth"
	"github.com/uddinArsalan/ferry-registry/adapters/password"
	"github.com/uddinArsalan/ferry-registry/adapters/token"
	"github.com/uddinArsalan/ferry-registry/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthService struct {
	authRepo      *repository.AuthRepo
	passwordStore password.PasswordStore
	tokenStore    token.TokenStore
	genauth.UnimplementedAuthServiceServer
}

var (
	ErrInvalidCredentials = status.Error(
		codes.Unauthenticated,
		"invalid crendetials",
	)
	ErrUnauthenticated = status.Error(
		codes.Unauthenticated,
		"unauthenticated",
	)
	ErrUserNotFound = status.Error(
		codes.NotFound,
		"user not found",
	)
	ErrInternalServer = status.Error(
		codes.Internal,
		"internal server error",
	)
	ErrInvalidInput = status.Error(
		codes.InvalidArgument,
		"invalid input",
	)
	AccessTokenTTL  = 15 * time.Minute
	RefreshTokenTTL = 7 * 24 * time.Hour
)

func NewAuthService(authRepo *repository.AuthRepo, passwordStore password.PasswordStore, tokenStore token.TokenStore) *AuthService {
	return &AuthService{
		authRepo:      authRepo,
		passwordStore: passwordStore,
		tokenStore:    tokenStore,
	}
}

func (a *AuthService) Register(ctx context.Context, req *genauth.RegisterRequest) (*genauth.RegisterResponse, error) {
	if req.Name == "" || req.Email == "" {
		return nil, ErrInvalidCredentials
	}
	hash, err := a.passwordStore.HashAndEncodePassword(req.Password)
	// need to consider something for error
	if err != nil {
		log.Printf("Err hashing or encoding password %v", err.Error())
		return nil, ErrInternalServer
	}
	userID, err := a.authRepo.CreateUser(ctx, req.Name, req.Email, hash)
	if err != nil {
		log.Printf("Err creating user %v", err.Error())
		return nil, ErrInternalServer
	}
	accessToken, err := a.tokenStore.GenerateToken(userID, AccessTokenTTL)
	if err != nil {
		log.Printf("Err generating token %v", err.Error())
		return nil, ErrInternalServer
	}
	refreshToken, err := a.tokenStore.GenerateRefreshToken()
	if err != nil {
		log.Printf("Err generating refresh token %v", err.Error())
		return nil, ErrInternalServer
	}
	now := time.Now()

	accessTokenExpiresAt := now.Add(AccessTokenTTL)
	refreshTokenExpiresAt := now.Add(RefreshTokenTTL)

	if err = a.authRepo.CreateRefreshToken(ctx, userID, a.tokenStore.HashToken(refreshToken), refreshTokenExpiresAt); err != nil {
		log.Printf("Err creating refresh token %v", err.Error())
		return nil, ErrInternalServer
	}
	return &genauth.RegisterResponse{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		AccessExpiresAt:  accessTokenExpiresAt.UnixMilli(),
		RefreshExpiresAt: refreshTokenExpiresAt.UnixMilli(),
	}, nil
}

func (a *AuthService) Login(ctx context.Context, req *genauth.LoginRequest) (*genauth.LoginResponse, error) {
	if req.Email == "" {
		return nil, ErrInvalidCredentials
	}
	user, err := a.authRepo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		log.Printf("Err fetching user %v", err.Error())
		return nil, ErrInternalServer
	}
	//compare password
	match, err := a.passwordStore.ComparePasswordAndHash(req.Password, user.PasswordHash)
	if err != nil {
		log.Printf("Err in compare and hash password %v", err.Error())
		return nil, ErrInternalServer
	}
	if !match {
		log.Println("password doesnt match")
		return nil, ErrInvalidCredentials
	}

	accessToken, err := a.tokenStore.GenerateToken(user.ID, AccessTokenTTL)
	if err != nil {
		log.Printf("Err generating token %v", err.Error())
		return nil, ErrInternalServer
	}
	refreshToken, err := a.tokenStore.GenerateRefreshToken()
	if err != nil {
		log.Printf("Err generating refresh token %v", err.Error())
		return nil, ErrInternalServer
	}
	now := time.Now()

	accessTokenExpiresAt := now.Add(AccessTokenTTL)
	refreshTokenExpiresAt := now.Add(RefreshTokenTTL)

	if err = a.authRepo.CreateRefreshToken(ctx, user.ID, a.tokenStore.HashToken(refreshToken), refreshTokenExpiresAt); err != nil {
		log.Printf("Err creating refresh token %v", err.Error())
		return nil, ErrInternalServer
	}
	return &genauth.LoginResponse{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		AccessExpiresAt:  accessTokenExpiresAt.UnixMilli(),
		RefreshExpiresAt: refreshTokenExpiresAt.UnixMilli(),
	}, nil
}

func (a *AuthService) Refresh(ctx context.Context, req *genauth.RefreshRequest) (*genauth.RefreshResponse, error) {
	if req.RefreshToken == "" {
		return nil, ErrInvalidInput
	}
	oldToken, err := a.authRepo.GetRefreshToken(ctx, a.tokenStore.HashToken(req.RefreshToken))
	if err != nil {
		return nil, ErrInternalServer
	}
	if oldToken.RevokedAt != nil {
		return nil, ErrUnauthenticated
	}
	if time.Now().After(oldToken.ExpiresAt) {
		return nil, ErrUnauthenticated
	}
	accessToken, err := a.tokenStore.GenerateToken(oldToken.UserID, AccessTokenTTL)
	if err != nil {
		return nil, ErrInternalServer
	}
	newToken, err := a.tokenStore.GenerateRefreshToken()
	if err != nil {
		return nil, ErrInternalServer
	}
	now := time.Now()

	accessTokenExpiresAt := now.Add(AccessTokenTTL)
	refreshTokenExpiresAt := now.Add(RefreshTokenTTL)

	if err = a.authRepo.CreateAndUpdateRefreshToken(ctx, oldToken.ID, a.tokenStore.HashToken(newToken), oldToken.UserID, refreshTokenExpiresAt); err != nil {
		return nil, ErrInternalServer
	}
	return &genauth.RefreshResponse{
		AccessToken:      accessToken,
		RefreshToken:     newToken,
		AccessExpiresAt:  accessTokenExpiresAt.UnixMilli(),
		RefreshExpiresAt: refreshTokenExpiresAt.UnixMilli(),
	}, nil
}
