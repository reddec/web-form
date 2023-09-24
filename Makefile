LOCAL := $(PWD)/.local
export PATH := $(LOCAL)/bin:$(PATH)
export GOBIN := $(LOCAL)/bin

LINTER := $(GOBIN)/golangci-lint
GORELEASER := $(GOBIN)/goreleaser

$(LINTER):
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.54.2

$(GORELEASER):
	go install github.com/goreleaser/goreleaser@v1.21.0

lint: $(LINTER)
	$(LINTER) run
.PHONY: lint

snapshot: $(GORELEASER)
	$(GORELEASER) release --snapshot --clean
	docker tag ghcr.io/reddec/$(notdir $(CURDIR)):$$(jq -r .version dist/metadata.json)-amd64 ghcr.io/reddec/$(notdir $(CURDIR)):latest

local: $(GORELEASER)
	$(GORELEASER) release -f .goreleaser.local.yaml --clean

test:
	go test -v ./...

.PHONY: test

local-dev:
	docker compose stop
	docker compose rm -fv
	docker compose up -d

.PHONY: local-dev