.PHONY: run build test tidy vet migrate-up migrate-down seed swagger

run:
	go run ./cmd/api

build:
	CGO_ENABLED=0 go build -o bin/api ./cmd/api && CGO_ENABLED=0 go build -o bin/migrate ./cmd/migrate

test:
	go test ./...

vet:
	go vet ./...

tidy:
	go mod tidy

migrate-up:
	go run ./cmd/migrate -direction up

migrate-down:
	go run ./cmd/migrate -direction down -steps 1

seed:
	@echo "seed arrives in M2"

swagger:
	swag init -g cmd/api/main.go -o docs --parseDependency
