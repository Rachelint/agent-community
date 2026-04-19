# agent-community

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A local-first collaboration hub for dispatching work to coding agents.

Chat with an agent to shape ideas into issues, assign issues to
worker agents, and review results via a GitHub-style mailbox.

## Status

Early scaffolding. Only `/healthz` works; data model, plugin system, and
UI are stubs.

## Architecture (one-page)

Three UI sections, all scoped per project:

- **Chat** — topic-based chat rooms. Each requirement starts a topic; you
  discuss approach with a chat agent, then publish the outcome as linked
  issues. Archive/delete topics when done.
- **Issues** — GitHub-style issues with flexible labels, parent/child
  links, and assignee = a worker agent. Dispatching an issue launches
  the worker.
- **Mailbox** — worker completion notifications. Jump from a
  notification back to the originating topic to review.

### Agents (chat + worker are symmetric)

Both chat and worker agents are external processes described by a JSON
manifest under `agents/<name>/agent.json`:

```json
{
  "name": "chat-codebuddy",
  "kind": "chat",
  "command": "codebuddy",
  "args": ["--print", "-y"],
  "cwd": "${topic.dir}",
  "protocol": "jsonrpc-stdio"
}
```

The host spawns the process, verifies startup, then steps back. Agents
drive everything else via JSON-RPC over stdio:

- `chat.delta` / `chat.done` for streaming chat replies
- `worker.log` / `worker.complete` for execution progress

Actor model: the host does not poll. A crashed worker becomes `orphan`
on the next manager sweep.

### Storage

- **SQLite** (via sqlc) holds all relational state: projects, agent
  members, topics, messages, issues, labels, comments, worker_runs,
  notifications.
- **Filesystem** holds bulky content: per-run workspace (git worktree,
  prompt, artifacts, raw stdout/stderr/events logs).

### Stack

| Layer    | Choice                                                   |
| -------- | -------------------------------------------------------- |
| Server   | Go 1.24 + Gin + sqlc + SQLite (modernc.org/sqlite)       |
| Realtime | WebSocket + JSON Patch streams (per-channel)             |
| UI       | React 18 + Vite + TS, TanStack Router/Query, shadcn/ui   |
| Theme    | GitHub-style (light, neutral grays, subtle borders)      |
| Bundle   | `go:embed` web/dist into a single binary                 |

## Layout

```
agent-community/
├── cmd/server/          # main.go
├── internal/
│   ├── api/             # HTTP handlers (Gin)
│   ├── config/          # config loading
│   ├── hub/             # websocket hub
│   ├── plugin/          # manifest + JSON-RPC + process manager
│   ├── service/         # domain services
│   └── store/           # sqlc-generated queries
├── db/
│   ├── migrations/      # SQL migrations
│   └── queries/         # sqlc input
├── web/                 # React SPA
├── agents/              # sample agent manifests
└── scripts/             # dev helpers
```

## Dev

Prereqs: Go 1.24+, Node 20+, npm (or pnpm).

```sh
make deps     # install web deps
make dev      # run server (:8080) + vite dev (:5173)
make build    # build single binary with embedded UI
```

Server listens on `:8080`. In dev the Vite server at `:5173` proxies
`/api` and `/ws` to it.

## Roadmap

See `ROADMAP.md` for phase plan. Current phase: **1 — data layer**.
