# Splitty Backend

Go + GraphQL backend for the Splitty expense-splitting app.

## Prerequisites

- Go 1.24+
- Docker (with Compose V2)

## Getting Started

```bash
# Start local Postgres
make docker-up

# Create the splitty_dev database (first time only)
make db-create

# Generate GraphQL code (required before first build)
make gqlgen

# Build
cd backend && go build ./...

# Run the server
make run

# Run tests
make test

# Stop Postgres
make docker-down
```

## Project Structure

```
backend/
  cmd/server/         # Server entry point
  internal/
    auth/             # Authentication (JWT, Apple Sign-In, passcode)
    config/           # Environment-based configuration
    db/               # Database connection and migrations
  graph/
    schema.graphqls   # GraphQL schema definition
    generated.go      # Generated runtime (not committed, run make gqlgen)
    model/            # Generated Go types (not committed)
    resolver.go       # Resolver struct with dependencies
    *.resolvers.go    # Resolver implementations
  gqlgen.yml          # gqlgen configuration
```

## Configuration

Environment variables:

| Variable | Description | Default |
|---|---|---|
| `SPLITTY_ENV` | `development` or `production` | `development` |
| `DATABASE_URL` | Postgres connection string | — |
| `JWT_PRIVATE_KEY` | RSA private key for JWT signing | — |

Local Postgres URL: `postgres://postgres@localhost:5432/splitty_dev?sslmode=disable`

## GraphQL

Schema files live in `graph/`. After editing, regenerate code:

```bash
make gqlgen
```

This runs `gqlgen generate` and outputs Go code to `graph/generated.go` and `graph/model/models_gen.go`.

The GraphQL playground is available at `http://localhost:8080/` when the server is running.

## Workflow
- Write tests for new features and bug fixes
- Run tests before creating a PR and before pushing changes to a PR
