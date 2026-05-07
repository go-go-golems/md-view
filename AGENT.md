# Agent Guidelines for go-go-golems/md-view

## Build Commands

- Run: `go run ./cmd/md-view`
- Build: `go build -o md-view ./cmd/md-view` or `make build`
- Test: `go test ./...` or `make test`
- Run single test: `go test ./pkg/path/to/package -run TestName`
- Generate: `go generate ./...`
- Lint: `make lint`
- Format: `go fmt ./...`
- GoReleaser snapshot: `make goreleaser`

IMPORTANT: To run the server and do some interaction with it, use tmux, this makes it very easy to kill a server.
Use capture-pane to read the output.

## Project Structure

- `cmd/md-view/`: CLI entry point
- `pkg/commands/`: Cobra commands (view, serve, stop, status)
- `pkg/daemon/`: Daemon management (start, stop, PID/socket files)
- `pkg/protocol/`: Unix socket JSON protocol between CLI and daemon
- `pkg/renderer/`: Markdown → HTML rendering (goldmark, chroma, mermaid, go:embed static)
- `pkg/server/`: HTTP server (routes, SSE live reload)
- `pkg/watcher/`: File system watcher for live reload
- `docs/`: User documentation
- `ttmp/`: Temporary documentation and debugging logs

<runningProcessesGuidelines>
- When testing TUIs, use tmux and capture-pane to interact with the UI.
- When using tmux, try to batch as many commands as possible when using send-keys.
- When running long-running processes (servers, etc...), use tmux to more easily interact and kill them.
- Kill a process using port $PORT: `lsof-who -p $PORT -k`. When building a web server, ALWAYS use this command to kill the process.
</runningProcessesGuidelines>

<goGuidelines>
- When implementing go interfaces, use the var _ Interface = &Foo{} to make sure the interface is always implemented correctly.
- Always use a context argument when appropriate.
- Use cobra for command-line applications.
- Use the "defaults" package name, instead of "default" package name, as it's reserved in go.
- Use github.com/pkg/errors for wrapping errors.
- When starting goroutines, use errgroup.
- Only use the toplevel go.mod, don't create new ones.
- When writing a new experiment / app, add zerolog logging to help debug and figure out how it works, add --log-level flag to set the log level.
- When using go:embed, import embed as `_ "embed"`
- When using build tagged features, make sure the software compiles without the tag as well
</goGuidelines>

<debuggingGuidelines>
If me or you the LLM agent seem to go down too deep in a debugging/fixing rabbit hole in our conversations, remind me to take a breath and think about the bigger picture instead of hacking away. Say: "I think I'm stuck, let's TOUCH GRASS". IMPORTANT: Don't try to fix errors by yourself more than twice in a row. Then STOP. Don't do anything else.
</debuggingGuidelines>

<generalGuidelines>
Don't add backwards compatibility layers or adapters unless explicitly asked. If you think there is a need for a backwards compatibility or adapting to an existing interface, STOP AND ASK ME IF THAT IS NECESSARY. Usually, I don't need backwards compatibility.

If it looks like your edits aren't applied, stop immediately and say "STOPPING BECAUSE EDITING ISN'T WORKING".
</generalGuidelines>
