.PHONY: all build run test lint lintmax golangci-lint-install gosec govulncheck goreleaser clean install dev tag-major tag-minor tag-patch release bump-glazed gifs

all: build

BINARY := md-view
VERSION=v0.1.0

GORELEASER_ARGS ?= --skip=sign --snapshot --clean
GORELEASER_TARGET ?= --single-target
GOLANGCI_LINT_VERSION ?= $(shell cat .golangci-lint-version)
GOLANGCI_LINT_BIN ?= $(CURDIR)/.bin/golangci-lint
GOLANGCI_LINT_ARGS ?= --timeout=5m ./cmd/... ./pkg/...
LINT_DIRS := $(shell git ls-files '*.go' | grep -vE '(^|/)ttmp/|(^|/)testdata/' | xargs -r -n1 dirname | sed 's#^#./#' | sort -u)
GOSEC_EXCLUDE_DIRS := -exclude-dir=.history -exclude-dir=testdata -exclude-dir=ttmp

build:
	GOWORK=off go generate ./...
	GOWORK=off go build -o $(BINARY) ./cmd/md-view

run: build
	./$(BINARY) view $(FILE)

test:
	GOWORK=off go test ./...

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

bump-glazed:
	GOWORK=off go get github.com/go-go-golems/glazed@latest
	GOWORK=off go get github.com/go-go-golems/clay@latest
	GOWORK=off go mod tidy

install: build
	GOWORK=off go build -o ./dist/md-view ./cmd/md-view && \
		cp ./dist/md-view $(shell which md-view 2>/dev/null || echo /usr/local/bin/md-view)

# Development: run server in foreground on a fixed port
dev: build
	./$(BINARY) serve --port 18765

clean:
	rm -f $(BINARY)
	rm -f ~/.local/state/md-view/md-view.pid
	rm -f ~/.local/state/md-view/md-view.sock
	rm -f ~/.local/state/md-view/md-view.port
