ifneq (,$(wildcard .env))
include .env
export $(shell sed 's/=.*//' .env)
endif

.PHONY: fmt lint test test-unit test-integ cover cover-html build up down

COVER_PROFILE ?= coverage.out

fmt:
	gofmt -w cmd internal pkg

lint:
	golangci-lint run

test: test-unit test-integ

test-unit:
	go test -race $(shell go list ./internal/... | grep -v '/internal/repository')

test-integ:
	go test -timeout=3m ./internal/repository/...

cover:
	go test -race -coverpkg=./internal/... -coverprofile=$(COVER_PROFILE) ./...
	go tool cover -func=$(COVER_PROFILE)

cover-html: cover
	go tool cover -html=$(COVER_PROFILE)

build:
	mkdir -p bin
	go build -trimpath -o bin/quotes ./cmd/quotes

up:
	docker compose up -d --wait postgres
	goose up
	docker compose up --build app

down:
	docker compose down

down-vol:
	docker compose down -v