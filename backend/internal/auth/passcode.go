package auth

import (
	"context"
	"log/slog"
	"net/mail"
	"strings"

	splittyv1 "github.com/kylejs/splitty/backend/gen/splitty/v1"
	"github.com/kylejs/splitty/backend/internal/config"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuthServer implements the gRPC AuthService handlers for passcode-based auth.
type AuthServer struct {
	splittyv1.UnimplementedAuthServiceServer
	env    string
	users  UserStore
	tokens TokenIssuer
}

// NewAuthServer creates an AuthServer with the given dependencies.
func NewAuthServer(env string, users UserStore, tokens TokenIssuer) *AuthServer {
	return &AuthServer{
		env:    env,
		users:  users,
		tokens: tokens,
	}
}

func (s *AuthServer) SendPasscode(_ context.Context, req *splittyv1.SendPasscodeRequest) (*splittyv1.SendPasscodeResponse, error) {
	if s.env != config.EnvDevelopment {
		return nil, status.Error(codes.Unavailable, "email passcode is not available")
	}

	email, err := normalizeEmail(req.GetEmail())
	if err != nil {
		return nil, err
	}

	slog.Info("passcode requested (any code accepted in dev mode)", "email", email)
	return &splittyv1.SendPasscodeResponse{}, nil
}

func (s *AuthServer) VerifyPasscode(ctx context.Context, req *splittyv1.VerifyPasscodeRequest) (*splittyv1.AuthResponse, error) {
	if s.env != config.EnvDevelopment {
		return nil, status.Error(codes.Unavailable, "email passcode is not available")
	}

	email, err := normalizeEmail(req.GetEmail())
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.GetCode()) == "" {
		return nil, status.Error(codes.InvalidArgument, "code is required")
	}

	user, err := s.users.UpsertByEmail(ctx, email)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to upsert user: %v", err)
	}

	accessToken, refreshToken, err := s.tokens.IssueTokens(ctx, user.ID, user.Email)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to issue tokens: %v", err)
	}

	return &splittyv1.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: &splittyv1.User{
			Id:          user.ID,
			Email:       user.Email,
			DisplayName: user.DisplayName,
		},
	}, nil
}

// normalizeEmail validates and normalizes an email address.
func normalizeEmail(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", status.Error(codes.InvalidArgument, "email is required")
	}
	addr, err := mail.ParseAddress(trimmed)
	if err != nil {
		return "", status.Error(codes.InvalidArgument, "invalid email address")
	}
	return strings.ToLower(addr.Address), nil
}
