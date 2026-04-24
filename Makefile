-include backend/.env
export

.PHONY: gqlgen docker-up docker-down db-create run test web-install web-codegen web-dev web-build

gqlgen:
	cd backend && go run github.com/99designs/gqlgen generate

docker-up:
	docker compose up -d

docker-down:
	docker compose down

db-create:
	@docker exec -i $$(docker compose ps -q postgres) psql -U postgres -tc "SELECT 1 FROM pg_database WHERE datname = 'splitty_dev'" | grep -q 1 || \
		docker exec -i $$(docker compose ps -q postgres) createdb -U postgres splitty_dev

run:
	cd backend && go run ./cmd/server

test:
	cd backend && go test ./...

web-install:
	cd web && npm install

web-codegen:
	cd web && npm run codegen

web-dev:
	cd web && npm run dev

web-build:
	cd web && npm run build
