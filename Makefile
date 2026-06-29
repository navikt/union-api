# Load secrets from .env if it exists (never committed).
# All variables — including those from .env — are exported to subprocesses,
# which means the uctl binary can resolve UNION_CLIENT_SECRET by name.
-include .env
export

CONFIG_FILE ?= config.local.yaml
IMAGE = europe-west1-docker.pkg.dev/nav-data-images-prod/dataplattform-infra/union-api
TAG ?= $(shell git rev-parse --short HEAD)

.PHONY: run build test docker-push

run:
	CONFIG_FILE=$(CONFIG_FILE) go run ./cmd/serviceaccounts

build:
	go build -o bin/serviceaccounts ./cmd/serviceaccounts

test:
	go test ./...

docker-push:
	docker build --platform linux/amd64 -t $(IMAGE):$(TAG) .
	docker push $(IMAGE):$(TAG)
