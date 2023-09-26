LOCAL := $(PWD)/.local
export PATH := $(LOCAL)/bin:$(PATH)
export GOBIN := $(LOCAL)/bin

MDIVERSION := 7.2.96
LINTER := $(GOBIN)/golangci-lint
GORELEASER := $(GOBIN)/goreleaser

$(LINTER):
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.54.2

$(GORELEASER):
	go install github.com/goreleaser/goreleaser@v1.21.0

lint: $(LINTER)
	$(LINTER) run
.PHONY: lint

# it will not download heavy eot and ttf
update-assets:
	mkdir -p internal/assets/static/css
	mkdir -p internal/assets/static/fonts
	curl -L -f -o internal/assets/static/css/materialdesignicons.min.css "https://cdn.jsdelivr.net/npm/@mdi/font@$(MDIVERSION)/css/materialdesignicons.min.css"
	cd internal/assets/static/css && cat materialdesignicons.min.css | \
		tr -s ';{},' '\n' | \
		grep url | \
		sed -rn 's|.*?url\("([^"]+?)".*|\1|p' | \
		grep -v '#' | \
		grep -v '.eot' | \
		grep -v '.ttf' | \
		cut -d '?' -f 1 | \
		xargs -I{} curl -L -f -o {} "https://cdn.jsdelivr.net/npm/@mdi/font@$(MDIVERSION)/css/{}"
.PHONY: update-assets


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