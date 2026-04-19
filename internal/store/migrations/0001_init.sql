-- 0001_init.sql
-- Phase 1 baseline: the two tables the rest of the system hangs off.
--
-- `projects` scopes everything in the product. Each project points at a
-- local git repo; worker runs will create worktrees under it later.
--
-- `agent_members` is the registry of installed chat / worker agents.
-- The table records what's present (discovered from agent.json manifests)
-- and the user's enable/disable preference. Runtime details (command,
-- args, cwd) are re-read from the manifest file when spawning, so the
-- DB never drifts from disk.

CREATE TABLE projects (
    id              TEXT    PRIMARY KEY,          -- UUIDv7
    name            TEXT    NOT NULL,
    repo_url        TEXT,                         -- git remote, optional
    repo_local      TEXT    NOT NULL,             -- absolute path to clone
    default_branch  TEXT    NOT NULL DEFAULT 'master',
    created_at      INTEGER NOT NULL,             -- unix millis
    updated_at      INTEGER NOT NULL
);

CREATE UNIQUE INDEX idx_projects_name ON projects(name);

CREATE TABLE agent_members (
    name            TEXT    PRIMARY KEY,          -- e.g. "chat-codebuddy"
    kind            TEXT    NOT NULL CHECK(kind IN ('chat','worker')),
    manifest_path   TEXT    NOT NULL,             -- absolute path to agent.json
    enabled         INTEGER NOT NULL DEFAULT 1,   -- 0/1
    registered_at   INTEGER NOT NULL,
    updated_at      INTEGER NOT NULL
);

-- Seed one stub of each kind so the frontend has something to render
-- during phase 1. Phase 3 replaces this with real manifest scanning.
INSERT INTO agent_members (name, kind, manifest_path, enabled, registered_at, updated_at) VALUES
    ('chat-codebuddy',   'chat',   'agents/chat-codebuddy/agent.json',   1, strftime('%s','now')*1000, strftime('%s','now')*1000),
    ('worker-codebuddy', 'worker', 'agents/worker-codebuddy/agent.json', 1, strftime('%s','now')*1000, strftime('%s','now')*1000);
