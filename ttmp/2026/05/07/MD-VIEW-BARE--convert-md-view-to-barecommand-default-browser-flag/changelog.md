# Changelog

## 2026-05-07

- Initial workspace created


## 2026-05-07

Converted all 4 commands to BareCommand, removed glazed sections, added --browser default firefox --new-window, added --no-browser flag (commit 757802b)

### Related Files

- /home/manuel/code/wesen/2026-05-07--md-server/pkg/commands/run.go — Passes browser/no-browser through protocol
- /home/manuel/code/wesen/2026-05-07--md-server/pkg/commands/serve.go — Converted to BareCommand
- /home/manuel/code/wesen/2026-05-07--md-server/pkg/commands/status.go — Converted to BareCommand
- /home/manuel/code/wesen/2026-05-07--md-server/pkg/commands/stop.go — Converted to BareCommand
- /home/manuel/code/wesen/2026-05-07--md-server/pkg/commands/view.go — Converted to BareCommand
- /home/manuel/code/wesen/2026-05-07--md-server/pkg/protocol/protocol.go — Added Browser field to Command
- /home/manuel/code/wesen/2026-05-07--md-server/pkg/server/server.go — Added openBrowserWith()

