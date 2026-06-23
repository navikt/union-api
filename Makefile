# Load secrets from .env if it exists (never committed).
# All variables — including those from .env — are exported to subprocesses,
# which means the uctl binary can resolve UNION_CLIENT_SECRET by name.
-include .env
export

CONFIG_FILE ?= config.local.yaml

.PHONY: run build test

run:
	CONFIG_FILE=$(CONFIG_FILE) go run ./cmd/serviceaccounts

build:
	go build -o bin/serviceaccounts ./cmd/serviceaccounts

test:
	go test ./...
