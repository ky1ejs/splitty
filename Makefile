.PHONY: proto-gen docker-up docker-down run test

proto-gen:
	cd backend && buf generate

docker-up:
	cd backend && docker compose up -d

docker-down:
	cd backend && docker compose down

run:
	cd backend && go run ./cmd/server

test:
	cd backend && go test ./...
