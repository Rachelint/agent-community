---
name: platform-ops
description: Operate the local agent-community host: start and stop the server, manage projects and issues, dispatch workers, inspect runs, and use the agent-facing plugin APIs.
---

# agent-community Platform Operations

Use this skill when the task is about operating the **agent-community platform itself** instead of implementing a feature inside a project repo.

## Use This Skill For

- Starting or stopping the local server
- Checking server health and logs
- Creating or listing projects
- Creating issues through the platform APIs
- Dispatching worker agents for an issue
- Inspecting, cancelling, or orphaning runs
- Calling the shared `/plugin/*` APIs from an external chat/worker agent

## Directory Convention

Shared platform skills live under:

```text
skills/<skill-name>/SKILL.md
```

Project contexts symlink this shared skill directory into:

```text
.codebuddy/skills
.claude/skills
```

## Quick Start

### Start the platform in development

```sh
make deps
make dev
```

- Server: `http://127.0.0.1:8080`
- Web UI: `http://127.0.0.1:5173`

### Start the production binary in the background

```sh
./scripts/start.sh
```

### Stop the background server

```sh
./scripts/stop.sh
```

### Health check

```sh
curl http://127.0.0.1:8080/healthz
```

## Runtime Knobs

Set these when you need a custom local environment:

```sh
AC_ADDR=:8080
AC_DATA_DIR=$HOME/.agent-community
AC_REPO_AGENTS_DIR=$(pwd)/agents
AC_CALLBACK_URL=http://127.0.0.1:8080
```

## User-Facing REST API

Use `/api/*` endpoints for UI-style operations.

### Projects

- `GET /api/projects`
- `POST /api/projects`
- `GET /api/projects/:pid`
- `DELETE /api/projects/:pid`

Typical create payload:

```json
{
  "name": "agent-community",
  "repo_local": "/absolute/path/to/repo",
  "default_branch": "main"
}
```

### Issues

- `GET /api/projects/:pid/issues`
- `POST /api/projects/:pid/issues`
- `GET /api/issues/:id`
- `PATCH /api/issues/:id`
- `DELETE /api/issues/:id`

### Worker dispatch and runs

- `POST /api/issues/:id/dispatch`
- `GET /api/issues/:id/runs`
- `GET /api/runs/:id`
- `GET /api/runs/:id/logs?stream=stdout|stderr|events`
- `POST /api/runs/:id/cancel`
- `POST /api/runs/:id/probe`
- `POST /api/runs/:id/mark_orphan`

Dispatch payload:

```json
{
  "plugin": "worker-name",
  "prompt": "What the worker should do"
}
```

## Agent-Facing Plugin API

Use `/plugin/*` endpoints for agent actions. Authenticate with:

```http
Authorization: Bearer <AC_PLUGIN_TOKEN>
```

### Project discovery

```http
GET /plugin/projects
```

Response:

```json
{
  "items": [
    { "id": "...", "name": "..." }
  ]
}
```

### Create an issue

```http
POST /plugin/projects/:pid/issues
```

Payload:

```json
{
  "title": "Issue title",
  "body": "Optional markdown body"
}
```

Success response is intentionally small:

```json
{
  "ok": true,
  "issue_id": "...",
  "issue_number": 42
}
```

Failure response is actionable:

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

### Worker callbacks

- `POST /plugin/runs/:id/ready`
- `POST /plugin/runs/:id/log`
- `POST /plugin/runs/:id/complete`
- `POST /plugin/runs/:id/fail`

## Standard Operating Flows

### 1. Bring the platform up

1. Run `make dev` for local development, or `./scripts/start.sh` for background mode.
2. Check `/healthz`.
3. Open the web UI if needed.

### 2. Register a project

1. Call `POST /api/projects` with an absolute `repo_local` path.
2. The repo must already be a git checkout.
3. The platform will initialize shared project context for supported agents.

### 3. Create and dispatch work

1. Create an issue in the project.
2. Dispatch a worker with `POST /api/issues/:id/dispatch`.
3. Track state via `GET /api/issues/:id/runs` and `GET /api/runs/:id`.
4. Read logs from `/api/runs/:id/logs`.

### 4. Let external agents create issues

1. Call `GET /plugin/projects`.
2. Choose a project id.
3. Call `POST /plugin/projects/:pid/issues`.
4. Keep only `issue_id` and `issue_number` in agent context.

## Troubleshooting

### Server does not start

- Check `bin/agent-community.log` when using `./scripts/start.sh`
- Ensure ports are free
- Verify `AC_REPO_AGENTS_DIR` if you moved the repo-managed agent assets

### Project creation fails

- `repo_local` must be an absolute path
- `repo_local` must already contain `.git`

### Dispatch fails

- Ensure the selected plugin exists, is enabled, and is a worker
- Check whether another run for the same issue is already active
- Inspect `/api/runs/:id/logs?stream=stderr`

### An active run is stuck

1. Call `POST /api/runs/:id/probe`
2. Inspect `stdout`, `stderr`, and `events`
3. If needed, call `cancel` or `mark_orphan`

### Plugin issue creation fails

- `unauthorized`: missing or wrong token
- `project_not_found`: refresh project list and retry
- `internal_error`: retry once, then surface the failure

## Guardrails

- Use `/api/*` for user/UI workflows.
- Use `/plugin/*` for agent workflows.
- Do not design new flows around chat-topic issue publishing.
- Prefer minimal agent-facing responses to avoid bloating context.
