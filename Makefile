.PHONY: all build run test lint lintmax golangci-lint-install gosec govulncheck goreleaser clean install dev tag-major tag-minor tag-patch release gifs frontend-css wails-dev wails-cli

# md-view is now a single Wails v2 desktop binary (MD-WAILS cutover).
# It is built with `wails build` (NOT plain `go build` — Wails injects build
# tags that a raw go build omits, causing a "will not build without the correct
# build tags" runtime error).
all: build

BINARY := md-view
VERSION := v0.2.0
WAILS_BUILD_TAGS ?= webkit2_41
UNAME_S := $(shell uname -s)

# Wails CLI selection. Newer Go toolchains (1.25+) break the
# golang.org/x/tools v0.30.0 vendored by Wails v2.12.0 during its binding
# static-analysis step ("internal error: package ... without types was
# imported from ..."). `make wails-cli` builds a patched CLI into .bin/wails;
# prefer it when present and otherwise fall back to one on PATH.
WAILS ?= $(if $(wildcard .bin/wails),.bin/wails,wails)
WAILS_VERSION ?= v2.12.0
WAILS_XTOOLS_VERSION ?= v0.50.0

# macOS Wails produces an .app bundle; Linux/Windows produce a bare binary.
ifeq ($(UNAME_S),Darwin)
APP_BINARY := build/bin/$(BINARY).app/Contents/MacOS/$(BINARY)
else
APP_BINARY := build/bin/$(BINARY)
endif

GORELEASER_ARGS ?= --skip=sign --snapshot --clean
GORELEASER_TARGET ?= --single-target
GOLANGCI_LINT_VERSION ?= $(shell cat .golangci-lint-version)
GOLANGCI_LINT_BIN ?= $(CURDIR)/.bin/golangci-lint
GOLANGCI_LINT_ARGS ?= --timeout=5m . ./cmd/... ./pkg/...
LINT_DIRS := $(shell git ls-files '*.go' | grep -vE '(^|/)ttmp/|(^|/)testdata/' | xargs -r -n1 dirname | sed 's|^|./|' | sort -u)
GOSEC_EXCLUDE_DIRS := -exclude-dir=.history -exclude-dir=testdata -exclude-dir=ttmp

# Build the frontend CSS (chroma.css + ui.css) before building the app.
# wails build embeds frontend/dist, so the generated CSS must be present.
build: frontend-css
	$(WAILS) build -tags $(WAILS_BUILD_TAGS) -s
ifeq ($(UNAME_S),Darwin)
# macOS needs the patched CLI (see wails-cli) for binding static-analysis.
build: .bin/wails
endif

# Build a patched Wails CLI into .bin/wails (everything stays inside the repo).
# Needed because the stock CLI's golang.org/x/tools cannot type-check packages
# with a newer Go toolchain; bumping x/tools fixes the static-analysis crash.
# `wails-cli` is a phony alias for the real file target.
wails-cli: .bin/wails

.bin/wails:
	@mkdir -p .bin .wails-cli
	cp -R "$$(GOWORK=off go env GOMODCACHE)/github.com/wailsapp/wails/v2@$(WAILS_VERSION)" .wails-cli/wails
	chmod -R u+w .wails-cli/wails
	cd .wails-cli/wails && GOWORK=off go mod edit -replace=golang.org/x/tools=golang.org/x/tools@$(WAILS_XTOOLS_VERSION)
	cd .wails-cli/wails && GOWORK=off go mod tidy
	cd .wails-cli/wails && GOWORK=off go build -o ../../.bin/wails ./cmd/wails
	@echo "Built patched Wails CLI: .bin/wails"

# Frontend assets: regenerate the static CSS the Wails frontend links (MD-WAILS DR-4).
# Produces frontend/dist/chroma.css (dual-theme code highlighting) and ui.css
# (frontmatter + button chrome, both themes).
frontend-css:
	GOWORK=off go run -tags $(WAILS_BUILD_TAGS) ./cmd/gen-chroma-css

# Development: hot-reload dev server (frontend + Go changes reload live).
wails-dev:
	$(WAILS) dev -tags $(WAILS_BUILD_TAGS)

run: build
	$(APP_BINARY) view $(FILE)

test:
	GOWORK=off go test -tags $(WAILS_BUILD_TAGS) ./...

golangci-lint-install:
	mkdir -p $(dir $(GOLANGCI_LINT_BIN))
	GOBIN=$(dir $(GOLANGCI_LINT_BIN)) GOWORK=off go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

lint: golangci-lint-install
	GOWORK=off $(GOLANGCI_LINT_BIN) config verify
	GOWORK=off $(GOLANGCI_LINT_BIN) run -v $(GOLANGCI_LINT_ARGS)

lintmax: golangci-lint-install
	GOWORK=off $(GOLANGCI_LINT_BIN) config verify
	GOWORK=off $(GOLANGCI_LINT_BIN) run -v --max-same-issues=100 $(GOLANGCI_LINT_ARGS)

gosec:
	GOWORK=off go install github.com/securego/gosec/v2/cmd/gosec@latest
	gosec -exclude-generated -exclude=G101,G304,G301,G306 $(GOSEC_EXCLUDE_DIRS) $(LINT_DIRS)

govulncheck:
	GOWORK=off go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

goreleaser:
	GOWORK=off goreleaser release $(GORELEASER_ARGS) $(GORELEASER_TARGET)

tag-major:
	git tag $(shell svu major)

tag-minor:
	git tag $(shell svu minor)

tag-patch:
	git tag $(shell svu patch)

release:
	git push origin --tags
	GOWORK=off GOPROXY=proxy.golang.org go list -m github.com/go-go-golems/md-view@$(shell svu current)

install: build
	cp $(APP_BINARY) $(shell which md-view 2>/dev/null || echo /usr/local/bin/md-view)

clean:
	rm -f $(BINARY)
	rm -rf build/bin
