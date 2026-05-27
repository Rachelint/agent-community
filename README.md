# agent-community

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A local-first collaboration hub for dispatching work to coding agents.

Chat with an agent to shape ideas into issues, assign issues to worker agents,
and review results via a GitHub-style mailbox.

## Status

Active local-first prototype. The server currently supports project management,
agent manifest scanning, labels, issues, chat topics/messages, worker dispatch,
run logs, plugin callbacks, and mailbox notifications. Realtime updates are
implemented with HTTP polling for now; WebSocket/streaming infrastructure remains
future work.

## Architecture

Three UI sections, all scoped per project:

- **Chat** — topic-based chat rooms for discussing requirements with a chat agent.
- **Issues** — GitHub-style issues with labels, single-level parent/child links,
  comments, and assignee = a worker agent. The issue body is a Markdown plan/spec
  and can be edited in the UI. Dispatching an issue launches the worker.
- **Mailbox** — worker lifecycle notifications. Jump from a notification back to
  the originating issue or run.

### Agents

Both chat and worker agents are external processes described by a JSON manifest
under `agents/<name>/agent.json` or `~/.agent-community/agents/<name>/agent.json`.
Bundled chat agents can load Markdown skills from `agents/<name>/skills/**/*.md`;
for example, `chat-codebuddy` has an issue-plan skill that explains how to edit
an issue's Markdown plan through the local API.

Example worker manifest:

```json
{
  "name": "worker-echo",
  "kind": "worker",
  "command": "bash",
  "args": ["${agent.dir}/entrypoint.sh"],
  "cwd": "${issue.workspace}/repo",
  "protocol": "jsonrpc-stdio"
}
```

The host spawns the process, sends `worker.start`, then steps back. Worker
state changes are delivered through authenticated HTTP callbacks; filesystem
artifacts are retained for logs and crash recovery.

### Worker protocol

A worker run receives context in two ways:

- Environment variables:
  - `AC_RUN_ID`
  - `AC_WORKSPACE`
  - `AC_RUN_DIR`
  - `AC_BRAIN_DIR`
  - `AC_CALLBACK_URL`
  - `AC_PLUGIN_TOKEN`
- A stdin JSON-RPC notification:

```json
{
  "jsonrpc": "2.0",
  "method": "worker.start",
  "params": {
    "run_id": "...",
    "workspace": "...",
    "run_dir": "...",
    "brain_dir": "...",
    "prompt": "...",
    "callback_url": "http://127.0.0.1:8080",
    "token": "..."
  }
}
```

After receiving `worker.start`, the worker reports lifecycle events through
the HTTP callback mailbox. stdout and stderr are raw diagnostic streams only.
Bundled workers use a Python3 deterministic runner (`lib/worker_runner/main.py`)
as the supervisor: the runner emits `ready`, phase events, `done.json`, and the
terminal callback, while the actual agent command runs as a child process.

Run files live under `~/.agent-community/workspaces/issues/<issue-id>/runs/<run-id>/`:

- `prompt.md` — persisted input prompt.
- `stdout.log` — captured stdout.
- `stderr.log` — captured stderr.
- `events.ndjson` — structured event log.
- `done.json` — crash recovery artifact, written by workers on exit and consumed only during server startup recovery:

```json
{
  "status": "completed|needs_review|failed",
  "exit_code": 0,
  "summary": "...",
  "mr_url": "..."
}
```

Worker callbacks and agent-facing plugin APIs are authenticated with `Authorization: Bearer $AC_PLUGIN_TOKEN`:

- `GET /plugin/projects` lists projects for agents as `{ "items": [{ "id": "...", "name": "..." }] }`
- `POST /plugin/projects/:pid/issues` creates an issue from `{ "title": "...", "body": "..." }` and returns a minimal success payload `{ "ok": true, "issue_id": "...", "issue_number": 42 }`
- `POST /plugin/runs/:id/ready` when the worker is ready to process the run
- `POST /plugin/runs/:id/log` with `{ "stream": "stdout|stderr|events", "data": "..." }`
- `POST /plugin/runs/:id/complete` with `{ "status": "completed|needs_review", "exit_code": 0, "summary": "...", "mr_url": "..." }`
- `POST /plugin/runs/:id/fail` with `{ "exit_code": 1, "summary": "..." }`

Plugin API failures use a small actionable envelope:

```json
{
  "ok": false,
  "error": {
    "code": "project_not_found",
    "message": "project not found",
    "hint": "Call GET /plugin/projects to refresh the project list, then retry with a valid project id."
  }
}
```

Phase events are newline-delimited JSON records in the `events` stream:

```json
{"event":"phase.started","phase":"agent"}
{"event":"phase.completed","phase":"agent","exit_code":0}
{"event":"phase.failed","phase":"agent","exit_code":1}
```

Normal state changes rely on callbacks. On server startup, the recovery pass
checks active runs once: `done.json` finishes runs whose terminal callback was
missed while the server was down; a dead process without `done.json` becomes
`orphan` so the user can inspect logs before deciding what to do next.

### Storage

- **SQLite** holds relational state: projects, agent members, topics, messages,
  issues, labels, comments, worker runs, and notifications.
- **Filesystem** holds bulky content: per-issue worktrees, per-run prompts,
  artifacts, stdout/stderr logs, event logs, and completion files.

### Stack

| Layer    | Choice                                                   |
| -------- | -------------------------------------------------------- |
| Server   | Go 1.25 + Gin + SQLite (modernc.org/sqlite, pure Go)     |
| Updates  | HTTP polling via React Query                             |
| UI       | React 18 + Vite + TypeScript + TanStack Query            |
| Theme    | GitHub-style light UI                                    |
| Bundle   | `go:embed` web/dist into a single binary                 |

## Layout

```text
agent-community/
├── cmd/server/          # main.go
├── internal/
│   ├── api/             # HTTP handlers (Gin)
│   ├── config/          # config loading
│   ├── hub/             # future realtime hub placeholder
│   ├── plugin/          # manifest + process manager
│   ├── reconcile/       # worker completion reconciler
│   └── store/           # sqlite driver, migrations, queries
│       └── migrations/  # embedded SQL migrations
├── web/                 # React SPA
├── agents/              # bundled agent manifests and wrappers
└── scripts/             # dev helpers
```

## Dev

Prereqs: Go 1.25+, Node 20+, npm.

```sh
make deps     # install web deps
make dev      # run server (:8080) + vite dev (:5173)
make test     # run Go tests
make check    # gofmt check + go test + go vet + web typecheck
make build    # build single binary with embedded UI
```

Server listens on `:8080`. In dev, Vite at `:5173` proxies `/api` to it.
Open the app at `http://127.0.0.1:5173`.

Run server and web separately if needed:

```sh
make server   # go run ./cmd/server, listens on :8080
make web      # vite dev server, listens on :5173
```

Normal background start/stop:

```sh
./scripts/start.sh    # builds, starts ./bin/agent-community in the background, then returns
./scripts/stop.sh     # stops the pid recorded by scripts/start.sh
```

`scripts/start.sh` writes process metadata under `bin/`:

```text
bin/agent-community.pid
bin/agent-community.log
```

Build and run the single binary in the foreground if you want logs in the terminal:

```sh
make build
./bin/agent-community
```

### Runtime data paths

By default, runtime data is stored under `~/.agent-community`:

```text
~/.agent-community/db.sqlite
~/.agent-community/workspaces/
~/.agent-community/agents/
~/.agent-community/topics/
```

Issue workspaces and run artifacts use this layout:

```text
~/.agent-community/workspaces/issues/<issue-id>/repo/
~/.agent-community/workspaces/issues/<issue-id>/runs/<run-id>/
```

A run directory may contain:

```text
prompt.md
stdout.log
stderr.log
events.ndjson
done.json
```

Bundled agent manifests are loaded from the repo `agents/` directory. User
agent manifests can be placed under `~/.agent-community/agents/<name>/agent.json`
and override bundled agents with the same name.

Runtime paths and ports can be overridden with environment variables:

```sh
AC_ADDR=:8080
AC_DATA_DIR=$HOME/.agent-community
AC_REPO_AGENTS_DIR=$(pwd)/agents
AC_CALLBACK_URL=http://127.0.0.1:8080
```

For an isolated normal run that keeps data inside the repo:

```sh
AC_DATA_DIR=/path/to/agent-community/.local-data ./scripts/start.sh
```

Useful direct checks:

```sh
go test ./...
go vet ./...
cd web && npm run typecheck
```
