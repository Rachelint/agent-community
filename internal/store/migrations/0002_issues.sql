-- 0002_issues.sql
-- Issues, labels, and comments. All entities are scoped by project_id
-- and cascade on project delete.
--
-- Design notes:
--   * issues.number is a project-local auto-increment. We maintain a
--     small helper table (project_issue_counter) and bump it inside the
--     same transaction as the INSERT, so two concurrent creates can't
--     collide.
--   * parent-child is a single level: parent_id must reference a
--     top-level issue (parent_id IS NULL). Enforced in the service
--     layer; we can't express it as a pure SQL constraint here.
--   * Soft delete: deleted_at column; UI only surfaces rows where
--     deleted_at IS NULL.
--   * Labels belong to a project; label.name is unique per project so
--     each project can choose its own taxonomy.

CREATE TABLE labels (
    id            TEXT    PRIMARY KEY,            -- UUIDv7
    project_id    TEXT    NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name          TEXT    NOT NULL,
    color         TEXT    NOT NULL,               -- hex without leading #
    description   TEXT,
    created_at    INTEGER NOT NULL,
    updated_at    INTEGER NOT NULL,
    UNIQUE (project_id, name)
);
CREATE INDEX idx_labels_project ON labels(project_id);

CREATE TABLE issues (
    id            TEXT    PRIMARY KEY,            -- UUIDv7
    project_id    TEXT    NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    number        INTEGER NOT NULL,               -- project-local, starts at 1
    parent_id     TEXT    REFERENCES issues(id) ON DELETE SET NULL,
    title         TEXT    NOT NULL,
    body          TEXT    NOT NULL DEFAULT '',    -- markdown
    status        TEXT    NOT NULL CHECK(status IN ('open','closed')) DEFAULT 'open',
    assignee      TEXT,                           -- agent_member.name (worker kind) or NULL
    created_at    INTEGER NOT NULL,
    updated_at    INTEGER NOT NULL,
    closed_at     INTEGER,
    deleted_at    INTEGER,                        -- soft delete
    UNIQUE (project_id, number)
);
CREATE INDEX idx_issues_project_status ON issues(project_id, status);
CREATE INDEX idx_issues_parent        ON issues(parent_id);
CREATE INDEX idx_issues_assignee      ON issues(assignee);

-- Per-project monotonically increasing counter for issue numbers.
-- One row per project; value stores the last handed-out number.
CREATE TABLE project_issue_counter (
    project_id    TEXT    PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
    last_number   INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE issue_labels (
    issue_id    TEXT NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    label_id    TEXT NOT NULL REFERENCES labels(id) ON DELETE CASCADE,
    PRIMARY KEY (issue_id, label_id)
);
CREATE INDEX idx_issue_labels_label ON issue_labels(label_id);

CREATE TABLE issue_comments (
    id          TEXT    PRIMARY KEY,
    issue_id    TEXT    NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    author      TEXT    NOT NULL,          -- 'user' | agent_member.name
    body        TEXT    NOT NULL,
    created_at  INTEGER NOT NULL
);
CREATE INDEX idx_issue_comments_issue ON issue_comments(issue_id, created_at);
