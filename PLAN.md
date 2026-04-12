# Splitty - Implementation Plan

Splitty is an iOS app + Go backend for splitting expenses between groups. This plan covers the full project scaffold and auth implementation.

---

## Auth Strategy

Two auth methods, same API surface in all environments:

| Method | Local (`SPLITTY_ENV=development`) | Production |
|---|---|---|
| **Sign in with Apple** | Works normally | Works normally |
| **Email + Passcode** | Accepts any passcode | Returns error (unavailable) |

### Sign in with Apple flow

```
iOS                              Go Backend                    Apple
 |-- SignInWithAppleButton tap ->|                              |
 |<-- identity token (JWT) -----------------------------------|
 |-- SignInWithApple RPC ------->|                              |
 |                               |-- Fetch JWKS, verify JWT -->|
 |                               |-- Upsert user in Postgres   |
 |                               |-- Issue access + refresh     |
 |<-- (access_token, refresh)   |                              |
```

### Email + Passcode flow

```
iOS                              Go Backend
 |-- SendPasscode(email) ------->|
 |                               |-- [dev] Log code to stdout, store in DB
 |                               |-- [prod] Return error: method unavailable
 |<-- OK (or error in prod)     |
 |                               |
 |-- VerifyPasscode(email,code)->|
 |                               |-- [dev] Accept any code, upsert user
 |                               |-- [prod] Return error: method unavailable
 |                               |-- Issue access + refresh
 |<-- (access_token, refresh)   |
```

### Token lifecycle

- **Access token**: JWT, RS256, 15 min TTL, contains `user_id` and `email`
- **Refresh token**: Opaque UUID, hashed (SHA-256) in Postgres, 90 day TTL
- **Refresh flow**: Client detects `Unauthenticated` gRPC status -> calls `RefreshToken` -> retries
- **Rotation**: Each refresh issues a new refresh token and invalidates the old one

---

## Tasks

### 1. Go module + project skeleton

Set up the Go module and directory structure.

**Files to create:**
- `backend/go.mod`
- `backend/cmd/server/main.go` (minimal, just a placeholder)
- `backend/internal/auth/` directory
- `backend/internal/config/config.go` (env-based config: `SPLITTY_ENV`, `DATABASE_URL`, `JWT_PRIVATE_KEY`)
- `backend/internal/db/` directory
- `Makefile` with targets: `proto-gen`, `docker-up`, `docker-down`, `run`, `test`

**Done when:** `go build ./...` succeeds from `backend/`

---

### 2. Docker Compose for local dev

Set up Postgres for local development.

**Files to create:**
- `backend/docker-compose.yml` — Postgres 16 container with volume, exposed on 5432

**Done when:** `docker-compose up -d` starts Postgres and it accepts connections

---

### 3. Proto definitions + buf config

Define the auth service protobuf and set up code generation.

**Files to create:**
- `backend/buf.yaml`
- `backend/buf.gen.yaml`
- `backend/proto/splitty/v1/auth.proto`
- `backend/proto/splitty/v1/splitty.proto` (stub — empty service for now)

**Proto schema:**

```protobuf
service AuthService {
  rpc SignInWithApple(SignInWithAppleRequest) returns (AuthResponse);
  rpc SendPasscode(SendPasscodeRequest) returns (SendPasscodeResponse);
  rpc VerifyPasscode(VerifyPasscodeRequest) returns (AuthResponse);
  rpc RefreshToken(RefreshTokenRequest) returns (AuthResponse);
}

message SignInWithAppleRequest {
  string identity_token = 1;
}

message SendPasscodeRequest {
  string email = 1;
}

message SendPasscodeResponse {}

message VerifyPasscodeRequest {
  string email = 1;
  string code = 2;
}

message RefreshTokenRequest {
  string refresh_token = 1;
}

message AuthResponse {
  string access_token = 1;
  string refresh_token = 2;
  User user = 3;
}

message User {
  string id = 1;
  string email = 2;
  string display_name = 3;
}
```

**Done when:** `make proto-gen` produces Go gRPC stubs in `backend/gen/`

---

### 4. Database migrations + connection pool

Set up the database layer.

**Files to create:**
- `backend/internal/db/db.go` — connection pool using `pgx/v5`
- `backend/internal/db/migrations/001_initial.sql`

**Schema:**

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    apple_sub TEXT UNIQUE,
    email TEXT,
    display_name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
```

**Done when:** Migration runs against local Postgres and tables exist

---

### 5. Auth — JWT token issuance and validation

Implement the session token layer that both auth methods share.

**Files to create/modify:**
- `backend/internal/auth/tokens.go`
  - `GenerateAccessToken(userID, email) -> (string, error)` — RS256 JWT, 15 min TTL
  - `ValidateAccessToken(tokenString) -> (Claims, error)`
  - `GenerateRefreshToken(userID) -> (string, error)` — opaque UUID, stores SHA-256 hash in Postgres
  - `ValidateRefreshToken(tokenString) -> (userID, error)` — looks up hash, checks expiry
  - `RotateRefreshToken(oldToken, userID) -> (newToken, error)` — invalidate old, issue new
- `backend/internal/auth/tokens_test.go`

**Done when:** Tests pass for token generation, validation, refresh, and rotation

---

### 6. Auth — Apple Sign-In verification

Implement Apple identity token verification.

**Files to create/modify:**
- `backend/internal/auth/apple.go`
  - Fetch Apple's JWKS from `https://appleid.apple.com/auth/keys` (with caching)
  - Verify JWT signature against JWKS
  - Validate claims: `iss` = `https://appleid.apple.com`, `aud` = app bundle ID, not expired
  - Extract `sub` (stable user ID) and `email`
- `backend/internal/auth/apple_test.go`

No third-party Apple library — write directly using `golang-jwt` + `net/http` (~50-100 lines).

**Done when:** Tests pass with a mock JWKS endpoint and test JWTs

---

### 7. Auth — Email passcode (env-aware)

Implement the passcode flow that varies by environment.

**Files to create/modify:**
- `backend/internal/auth/passcode.go`
  - `SendPasscode(email)`:
    - dev: log passcode to stdout (or accept any code, so sending is optional)
    - prod: return gRPC `Unavailable` error
  - `VerifyPasscode(email, code)`:
    - dev: accept any code, upsert user by email, return tokens
    - prod: return gRPC `Unavailable` error
- `backend/internal/auth/passcode_test.go`

**Done when:** Tests pass for both dev and prod behavior

---

### 8. Auth — gRPC interceptor

Wire up authentication middleware for all RPCs.

**Files to create/modify:**
- `backend/internal/auth/interceptor.go`
  - Unary and stream interceptors using `go-grpc-middleware/v2`
  - Extract bearer token from gRPC metadata
  - Verify JWT, inject `user_id` into context
  - Skip auth for: `SignInWithApple`, `SendPasscode`, `VerifyPasscode`, `RefreshToken`
- `backend/internal/auth/interceptor_test.go`

**Done when:** Tests pass — authed calls succeed, unauthed calls get `Unauthenticated`

---

### 9. gRPC server — wire everything up

Create the server entry point that ties all pieces together.

**Files to modify:**
- `backend/cmd/server/main.go`
  - Load config
  - Connect to Postgres, run migrations
  - Generate or load RSA key pair for JWT signing
  - Register `AuthService` with gRPC server
  - Attach auth interceptor
  - Serve on configured port

**Done when:** `docker-compose up` + `go run ./cmd/server` starts the server, and `grpcurl` can call `SendPasscode` + `VerifyPasscode` successfully in dev mode

---

### 10. iOS project setup

Scaffold the iOS app with xcodegen.

**Files to create:**
- `ios/project.yml` — xcodegen spec (deployment target, Swift packages for grpc-swift)
- `ios/Splitty/App/SplittyApp.swift` — app entry point
- `ios/Splitty/Services/GRPCClient.swift` — gRPC client setup, auth interceptor (attach bearer token from Keychain)
- `ios/Splitty/Keychain/KeychainHelper.swift` — store/retrieve tokens

**Done when:** `xcodegen generate` produces a buildable Xcode project

---

### 11. iOS auth — login screen

Build the login UI and auth flow.

**Files to create:**
- `ios/Splitty/Auth/LoginView.swift`
  - `SignInWithAppleButton` (always shown)
  - Email + passcode form (for dev — could be conditionally shown via a build config flag)
- `ios/Splitty/Auth/AuthViewModel.swift`
  - Auth state management (signed out, loading, signed in)
  - Call `SignInWithApple` / `SendPasscode` + `VerifyPasscode` RPCs
  - Store tokens in Keychain on success
  - Token refresh: detect `Unauthenticated` -> call `RefreshToken` -> retry

**Done when:** Login screen renders in SwiftUI preview, auth flow works against local backend in simulator

---

## Dependencies

### Go
| Dependency | Purpose |
|---|---|
| `github.com/golang-jwt/jwt/v5` | JWT creation/verification |
| `github.com/grpc-ecosystem/go-grpc-middleware/v2` | gRPC auth interceptor |
| `google.golang.org/grpc` | gRPC server |
| `google.golang.org/protobuf` | Protobuf runtime |
| `github.com/jackc/pgx/v5` | Postgres driver |
| `github.com/pressly/goose/v3` | Database migrations |

### iOS
| Dependency | Purpose |
|---|---|
| `grpc-swift` | gRPC client (SPM) |
| `AuthenticationServices` | Sign in with Apple (system framework) |

### Tools
| Tool | Purpose |
|---|---|
| `buf` | Proto compilation and linting |
| `xcodegen` | Xcode project generation |
| `docker` / `docker-compose` | Local Postgres + backend |
