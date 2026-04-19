# Roadmap

## Phase 0 — Skeleton (this commit)

- Repo layout, toolchain wired, `make dev` runs both sides
- Server exposes `/healthz`
- Web shows an "Agent Community" placeholder

## Phase 1 — Data layer + projects

- SQLite (modernc.org/sqlite) + embedded migration runner
- Tables: `projects`, `agent_members` (registry)
- REST: projects CRUD, agent_members list/reload
- Web: project switcher in sidebar

## Phase 2 — Issues domain

- Tables: `issues`, `labels`, `issue_labels`, `issue_comments`
- REST: issues/labels/comments CRUD, parent/child, filtering
- Web: Issues list + detail pages (no worker yet)

## Phase 3 — Agent runtime + worker dispatch

- `internal/plugin`: manifest loader, process manager, JSON-RPC stdio
- Tables: `worker_runs`
- REST: dispatch issue → spawn worker; agent callback API
  (`/plugin/runs/:id/log`, `/plugin/runs/:id/complete`)
- FS layout: `~/.agent-community/workspaces/<run_id>/`
- Minimal worker agent manifest + a reference shell wrapper

## Phase 4 — Mailbox + run viewer

- Table: `notifications`
- REST + WS for notifications and run log streaming
- Web: Mailbox page, Run detail page with log viewer

## Phase 5 — Chat (topics)

- Tables: `topics`, `topic_messages`
- Reuse agent runtime for `kind: "chat"`
- Web: Chat page with topic list + streaming message view

## Phase 6 — Topic ↔ Issue link + AI-drafted issues

- Table: `topic_issues`
- Action button in chat: chat agent emits structured issue draft → user
  confirms → posted to Issues and linked back to topic

## Phase 7 — Polish

- Keyboard shortcuts
- Log search + filtering
- Run rerun (continue with feedback)
- Agent manifest hot reload
- Single-binary packaging
