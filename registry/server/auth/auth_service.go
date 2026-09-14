package auth_service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/uddinArsalan/ferry-registry/adapters/password"
	"github.com/uddinArsalan/ferry-registry/adapters/token"
	genauth "github.com/uddinArsalan/ferry-registry/proto/auth"
	"github.com/uddinArsalan/ferry-registry/repository"
)

type AuthService struct {
	authRepo      *repository.AuthRepo
	passwordStore password.PasswordStore
	tokenStore    token.TokenStore
	genauth.UnimplementedAuthServiceServer
}

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized request")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidInput       = errors.New("invalid input")
	AccessTokenTTL        = 15 * time.Minute
	RefreshTokenTTL       = 7 * 24 * time.Hour
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
		return nil, ErrInvalidInput
	}
	hash, err := a.passwordStore.HashAndEncodePassword(req.Password)
	// need to consider something for error
	if err != nil {
		return nil, err
	}
	userID, err := a.authRepo.CreateUser(ctx, req.Name, req.Email, hash)
	if err != nil {
		return nil, err
	}
	accessToken, err := a.tokenStore.GenerateToken(userID, AccessTokenTTL)
	if err != nil {
		return nil, err
	}
	refreshToken, err := a.tokenStore.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}
	now := time.Now()

	accessTokenExpiresAt := now.Add(AccessTokenTTL)
	refreshTokenExpiresAt := now.Add(RefreshTokenTTL)

	if err = a.authRepo.CreateRefreshToken(ctx, userID, a.tokenStore.HashToken(refreshToken), refreshTokenExpiresAt); err != nil {
		return nil, err
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
			// need to do something here so ferry client knows this user doesnt exist
			// and show a message to user to create instead
		}
		return nil, err
	}
	//compare password
	match, err := a.passwordStore.ComparePasswordAndHash(req.Password, user.PasswordHash)
	if err != nil {
		return nil, err
	}
	if !match {
		return nil, ErrInvalidCredentials
	}

	accessToken, err := a.tokenStore.GenerateToken(user.ID, AccessTokenTTL)
	if err != nil {
		return nil, err
	}
	refreshToken, err := a.tokenStore.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}
	now := time.Now()

	accessTokenExpiresAt := now.Add(AccessTokenTTL)
	refreshTokenExpiresAt := now.Add(RefreshTokenTTL)

	if err = a.authRepo.CreateRefreshToken(ctx, user.ID, a.tokenStore.HashToken(refreshToken), refreshTokenExpiresAt); err != nil {
		return nil, err
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
		return nil, ErrUnauthorized
	}
	oldToken, err := a.authRepo.GetRefreshToken(ctx, a.tokenStore.HashToken(req.RefreshToken))
	if err != nil {
		return nil, err
	}
	if oldToken.RevokedAt != nil {
		return nil, ErrUnauthorized
	}
	if time.Now().After(oldToken.ExpiresAt) {
		return nil, ErrUnauthorized
	}
	accessToken, err := a.tokenStore.GenerateToken(oldToken.UserID, AccessTokenTTL)
	if err != nil {
		return nil, err
	}
	newToken, err := a.tokenStore.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}
	now := time.Now()

	accessTokenExpiresAt := now.Add(AccessTokenTTL)
	refreshTokenExpiresAt := now.Add(RefreshTokenTTL)

	if err = a.authRepo.CreateAndUpdateRefreshToken(ctx, oldToken.ID, a.tokenStore.HashToken(newToken), oldToken.UserID, refreshTokenExpiresAt); err != nil {
		return nil, err
	}
	return &genauth.RefreshResponse{
		AccessToken:      accessToken,
		RefreshToken:     newToken,
		AccessExpiresAt:  accessTokenExpiresAt.UnixMilli(),
		RefreshExpiresAt: refreshTokenExpiresAt.UnixMilli(),
	}, nil
}
