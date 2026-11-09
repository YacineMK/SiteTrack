.PHONY: run build test tidy migrate

DATABASE_URL ?= postgres://sitetrack:sitetrack@localhost:5432/sitetrack?sslmode=disable
DBMATE_IMAGE ?= ghcr.io/amacneil/dbmate:latest

run:
	go run ./cmd

build:
	go build -o bin/sitetrack ./cmd

test:
	go test ./... -count=1

tidy:
	go mod tidy

migrate:
	docker run --rm --network host -v "$(CURDIR)/migrations:/db/migrations" -e DATABASE_URL="$(DATABASE_URL)" $(DBMATE_IMAGE) --no-dump-schema up
