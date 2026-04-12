.PHONY: proto-gen docker-up docker-down db-create run test

proto-gen:
	cd backend && buf generate

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
