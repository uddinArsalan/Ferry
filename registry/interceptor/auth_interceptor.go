package interceptor

import (
	"context"
	"log"
	"strconv"
	"strings"

	"github.com/uddinArsalan/ferry-registry/adapters/token"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var (
	errMissingMetadata = status.Error(codes.InvalidArgument, "missing metadata")
	errInvalidToken    = status.Error(codes.Unauthenticated, "invalid token")
	ErrUnauthenticated = status.Error(codes.Unauthenticated, "unauthenticated")
)

type AuthInterceptor struct {
	tokenStore token.TokenStore
}

func NewAuthInterceptor(tokenStore token.TokenStore) AuthInterceptor {
	return AuthInterceptor{
		tokenStore: tokenStore,
	}
}

type UserID struct{}

func (a AuthInterceptor) UnaryInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	if strings.Contains(info.FullMethod, "/proto.auth") {
		return handler(ctx, req)
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errMissingMetadata
	}
	authorization := md["authorization"]
	if len(authorization) < 1 {
		return nil, errInvalidToken
	}
	token := strings.TrimPrefix(authorization[0], "Bearer ")
	claims, err := a.tokenStore.VerifyToken(token)
	if err != nil {
		log.Fatalf("invalid token %v", err.Error())
		return nil, errInvalidToken
	}
	ctxWithClaim := context.WithValue(ctx, UserID{}, claims.Subject)
	m, err := handler(ctxWithClaim, req)
	if err != nil {
		log.Fatalf("RPC failed with error: %v", err)
		return nil, err
	}
	return m, nil
}

func GetUserIDFromContext(ctx context.Context) (int64, bool) {
	userID := ctx.Value(UserID{})
	if userID == nil {
		return -1, false
	}
	userIDInt, err := strconv.ParseInt(userID.(string), 10, 64)
	if err != nil {
		return -1, false
	}
	return userIDInt, true
}
