# Splitty Backend

Go + gRPC backend for the Splitty expense-splitting app.

## Prerequisites

- Go 1.24+
- Docker (with Compose V2)
- [buf](https://buf.build/) for protobuf code generation

## Getting Started

```bash
# Start local Postgres
make docker-up

# Generate proto stubs (required before first build)
make proto-gen

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
  gen/splitty/v1/     # Generated gRPC stubs (not committed, run make proto-gen)
  proto/splitty/v1/   # Protobuf definitions
  buf.yaml            # buf module config
  buf.gen.yaml        # buf code generation config
  docker-compose.yml  # Local Postgres
```

## Configuration

Environment variables:

| Variable | Description | Default |
|---|---|---|
| `SPLITTY_ENV` | `development` or `production` | `development` |
| `DATABASE_URL` | Postgres connection string | — |
| `JWT_PRIVATE_KEY` | RSA private key for JWT signing | — |

Local Postgres URL: `postgres://splitty:splitty@localhost:5432/splitty_dev?sslmode=disable`

## Protobuf

Proto files live in `proto/splitty/v1/`. After editing, regenerate stubs:

```bash
make proto-gen
```

This runs `buf generate` and outputs Go code to `gen/splitty/v1/`.
