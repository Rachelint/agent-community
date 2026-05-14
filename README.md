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

- **Chat** — topic-based chat rooms. Discuss requirements with a chat agent,
  then publish the outcome as linked issues.
- **Issues** — GitHub-style issues with labels, single-level parent/child links,
  comments, and assignee = a worker agent. Dispatching an issue launches the
  worker.
- **Mailbox** — worker lifecycle notifications. Jump from a notification back to
  the originating issue or run.

### Agents

Both chat and worker agents are external processes described by a JSON manifest
under `agents/<name>/agent.json` or `~/.agent-community/agents/<name>/agent.json`:

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

The host spawns the process, verifies startup with `worker.ready`, then tracks
run state through callbacks and filesystem artifacts.

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

Startup:

```json
{"jsonrpc":"2.0","method":"worker.ready","params":{}}
```

Run files live under `~/.agent-community/workspaces/issues/<issue-id>/runs/<run-id>/`:

- `prompt.md` — persisted input prompt.
- `stdout.log` — captured stdout.
- `stderr.log` — captured stderr.
- `events.ndjson` — structured event log.
- `done.json` — reconciler fallback result:

```json
{
  "status": "completed|needs_review|failed",
  "exit_code": 0,
  "summary": "...",
  "mr_url": "..."
}
```

Preferred plugin callbacks are authenticated with `Authorization: Bearer $AC_PLUGIN_TOKEN`:

- `POST /plugin/runs/:id/log` with `{ "stream": "stdout|stderr|events", "data": "..." }`
- `POST /plugin/runs/:id/complete` with `{ "status": "completed|needs_review", "exit_code": 0, "summary": "...", "mr_url": "..." }`
- `POST /plugin/runs/:id/fail` with `{ "exit_code": 1, "summary": "..." }`

If a running worker exits without `done.json`, the reconciler marks the run as
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

Useful direct checks:

```sh
go test ./...
go vet ./...
cd web && npm run typecheck
```
