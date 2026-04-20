-- 0004_chat.sql
-- Chat topics and messages.

CREATE TABLE chat_topics (
    id          TEXT    PRIMARY KEY,
    project_id  TEXT    NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    title       TEXT    NOT NULL,
    status      TEXT    NOT NULL CHECK(status IN ('open','closed')) DEFAULT 'open',
    plugin      TEXT    NOT NULL,
    pid         INTEGER,
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);
CREATE INDEX idx_chat_topics_project ON chat_topics(project_id, updated_at DESC);

CREATE TABLE chat_messages (
    id          TEXT    PRIMARY KEY,
    topic_id    TEXT    NOT NULL REFERENCES chat_topics(id) ON DELETE CASCADE,
    role        TEXT    NOT NULL CHECK(role IN ('user','assistant','system')),
    content     TEXT    NOT NULL,
    in_reply_to TEXT,
    created_at  INTEGER NOT NULL
);
CREATE INDEX idx_chat_messages_topic ON chat_messages(topic_id, created_at);
