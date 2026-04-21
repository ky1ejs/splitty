package graph

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kylejs/splitty/backend/internal/auth"
	"github.com/kylejs/splitty/backend/internal/config"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require
// here.

type Resolver struct {
	Pool         *pgxpool.Pool
	TokenService *auth.TokenService
	Config       config.Config
}
