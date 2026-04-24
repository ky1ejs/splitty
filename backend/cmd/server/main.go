package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/kylejs/splitty/backend/graph"
	"github.com/kylejs/splitty/backend/internal/auth"
	"github.com/kylejs/splitty/backend/internal/config"
	"github.com/kylejs/splitty/backend/internal/db"
)

func main() {
	cfg := config.Load()
	fmt.Printf("splitty server starting in %s mode\n", cfg.Env)

	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx := context.Background()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	log.Println("database connected and migrations applied")

	var tokenService *auth.TokenService
	if cfg.JWTPrivateKey != "" {
		tokenService, err = auth.NewTokenService(cfg.JWTPrivateKey, pool)
		if err != nil {
			log.Fatalf("failed to create token service: %v", err)
		}
	} else if cfg.Env != config.EnvDevelopment {
		log.Fatal("JWT_PRIVATE_KEY is required in non-development environments")
	}

	var tokenIssuer auth.TokenIssuer
	if tokenService != nil {
		tokenIssuer = tokenService
	} else {
		tokenIssuer = &auth.DevTokenIssuer{}
	}

	userStore := auth.NewPgUserStore(pool)

	passcodeService := auth.NewPasscodeService(
		cfg.Env,
		userStore,
		tokenIssuer,
	)

	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{
		Resolvers: &graph.Resolver{
			Pool:            pool,
			TokenService:    tokenService,
			PasscodeService: passcodeService,
			UserStore:       userStore,
			Config:          cfg,
		},
	}))

	if cfg.Env == config.EnvDevelopment {
		http.Handle("/", playground.Handler("Splitty", "/query"))
	}

	var queryHandler http.Handler = srv
	if tokenService != nil {
		queryHandler = auth.Middleware(tokenService)(srv)
	}
	http.Handle("/query", queryHandler)

	port := "8080"
	if cfg.Env == config.EnvDevelopment {
		log.Printf("GraphQL playground at http://localhost:%s/", port)
	}
	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
