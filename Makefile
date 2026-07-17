GO ?= go
PNPM ?= pnpm
REGISTRY ?= ghcr.io/3011
VERSION ?= dev

.PHONY: fmt docs-check test test-enrollment build build-web check-web check images push compose-up compose-down

fmt:
	$(GO)fmt -w cmd internal

build:
	mkdir -p bin
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags="-s -w" -o bin/cfscan-server ./cmd/server
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags="-s -w -X main.version=$(VERSION)" -o bin/cfscan-agent ./cmd/agent

build-web:
	cd web && $(PNPM) build

docs-check:
	python3 scripts/check_docs.py

check-web:
	cd web && $(PNPM) check

test:
	$(GO) test ./...

test-enrollment:
	./scripts/test-agent-enrollment.sh

check: docs-check test build check-web

images: check
	docker build -f Dockerfile.server.runtime -t $(REGISTRY)/cfscan-server:$(VERSION) .
	docker build -f Dockerfile.agent.runtime -t $(REGISTRY)/cfscan-agent:$(VERSION) .
	docker build -f web/Dockerfile.runtime -t $(REGISTRY)/cfscan-web:$(VERSION) web

push: images
	docker push $(REGISTRY)/cfscan-server:$(VERSION)
	docker push $(REGISTRY)/cfscan-agent:$(VERSION)
	docker push $(REGISTRY)/cfscan-web:$(VERSION)

compose-up:
	docker compose --profile agent up -d --build

compose-down:
	docker compose --profile agent down
