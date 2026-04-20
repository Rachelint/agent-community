-- 0005_topic_issue_link.sql
-- Add source_topic_id to issues for Topic→Issue linking.
ALTER TABLE issues ADD COLUMN source_topic_id TEXT REFERENCES chat_topics(id) ON DELETE SET NULL;
CREATE INDEX idx_issues_source_topic ON issues(source_topic_id);
