.PHONY: build run test lint clean install

BINARY := md-view
GO := go

build:
	$(GO) build -o $(BINARY) ./cmd/md-view

run: build
	./$(BINARY) view $(FILE)

test:
	$(GO) test ./... -count=1 -v

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY)
	rm -f ~/.local/state/md-view/md-view.pid
	rm -f ~/.local/state/md-view/md-view.sock
	rm -f ~/.local/state/md-view/md-view.port

install: build
	$(GO) install ./cmd/md-view

# Development: run server in foreground on a fixed port
dev: build
	./$(BINARY) serve --port 18765
