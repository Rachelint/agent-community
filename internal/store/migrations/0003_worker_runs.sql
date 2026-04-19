-- 0003_worker_runs.sql
-- Worker runs and notifications.
--
-- A worker_run is one execution of a worker agent against an issue.
-- One issue can accumulate many runs (retries, re-dispatches). At any
-- given moment the service layer enforces "at most one running run
-- per issue" — if the worker crashes silently, the run stays in
-- 'running' until the user probes and marks it 'orphan', which is how
-- the slot is released without any heuristic-based auto-recovery.
--
-- Notifications are a simple per-project inbox. Phase 4 renders the
-- UI; phase 3 just writes rows so completion of a worker leaves a
-- trail the user can see later.

CREATE TABLE worker_runs (
    id             TEXT    PRIMARY KEY,                  -- UUIDv7
    issue_id       TEXT    NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    project_id     TEXT    NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    plugin         TEXT    NOT NULL,                     -- agent_members.name
    status         TEXT    NOT NULL CHECK(status IN (
                       'queued','running','needs_review',
                       'completed','failed','cancelled','orphan'
                   )),
    pid            INTEGER,                              -- OS pid while running
    workspace_dir  TEXT    NOT NULL,                     -- absolute path
    started_at     INTEGER,                              -- unix millis
    finished_at    INTEGER,
    exit_code      INTEGER,
    mr_url         TEXT,
    summary        TEXT,
    created_at     INTEGER NOT NULL
);
CREATE INDEX idx_worker_runs_issue   ON worker_runs(issue_id, created_at DESC);
CREATE INDEX idx_worker_runs_status  ON worker_runs(status);
CREATE INDEX idx_worker_runs_project ON worker_runs(project_id);

CREATE TABLE notifications (
    id           TEXT    PRIMARY KEY,
    project_id   TEXT    NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    kind         TEXT    NOT NULL,                -- 'worker_completed' | 'worker_failed' | ...
    issue_id     TEXT    REFERENCES issues(id) ON DELETE CASCADE,
    run_id       TEXT    REFERENCES worker_runs(id) ON DELETE CASCADE,
    title        TEXT    NOT NULL,
    body         TEXT,
    read_at      INTEGER,
    archived_at  INTEGER,
    created_at   INTEGER NOT NULL
);
CREATE INDEX idx_notifications_project_unread
    ON notifications(project_id, read_at, archived_at);
CREATE INDEX idx_notifications_created
    ON notifications(created_at DESC);
