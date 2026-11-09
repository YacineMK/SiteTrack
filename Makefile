.PHONY: run build test tidy migrate

DATABASE_URL ?= postgres://sitetrack:sitetrack@localhost:5432/sitetrack?sslmode=disable

run:
	go run ./cmd

build:
	go build -o bin/sitetrack ./cmd

test:
	go test ./... -count=1

tidy:
	go mod tidy

migrate:
	goose -dir migrations postgres "$(DATABASE_URL)" up
