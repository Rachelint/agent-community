package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// ChatTopic represents a chat conversation thread.
type ChatTopic struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	Plugin    string `json:"plugin"`
	PID       *int   `json:"pid,omitempty"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

// TopicCreate holds fields needed to insert a new topic.
type TopicCreate struct {
	ProjectID string
	Title     string
	Plugin    string
}

// ChatMessage represents a single message in a chat topic.
type ChatMessage struct {
	ID        string  `json:"id"`
	TopicID   string  `json:"topic_id"`
	Role      string  `json:"role"`
	Content   string  `json:"content"`
	InReplyTo *string `json:"in_reply_to,omitempty"`
	CreatedAt int64   `json:"created_at"`
}

// MessageCreate holds fields needed to insert a new message.
type MessageCreate struct {
	TopicID   string
	Role      string
	Content   string
	InReplyTo *string
}

// CreateTopic inserts a new open chat topic.
func (s *Store) CreateTopic(ctx context.Context, id string, in TopicCreate) (*ChatTopic, error) {
	now := time.Now().UnixMilli()
	if _, err := s.DB.ExecContext(ctx, `
		INSERT INTO chat_topics (id, project_id, title, status, plugin, created_at, updated_at)
		VALUES (?, ?, ?, 'open', ?, ?, ?)
	`, id, in.ProjectID, in.Title, in.Plugin, now, now); err != nil {
		return nil, err
	}
	return &ChatTopic{
		ID: id, ProjectID: in.ProjectID, Title: in.Title,
		Status: "open", Plugin: in.Plugin,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

// GetTopic fetches a topic by id.
func (s *Store) GetTopic(ctx context.Context, id string) (*ChatTopic, error) {
	row := s.DB.QueryRowContext(ctx, topicSelect+` WHERE id = ?`, id)
	return scanTopic(row.Scan)
}

// ListTopicsByProject returns topics for a project, newest first.
func (s *Store) ListTopicsByProject(ctx context.Context, projectID string) ([]ChatTopic, error) {
	rows, err := s.DB.QueryContext(ctx,
		topicSelect+` WHERE project_id = ? ORDER BY updated_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ChatTopic{}
	for rows.Next() {
		t, err := scanTopic(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

// UpdateTopicPID sets or clears the pid on a topic.
func (s *Store) UpdateTopicPID(ctx context.Context, id string, pid *int) error {
	now := time.Now().UnixMilli()
	var pidVal any
	if pid != nil {
		pidVal = *pid
	}
	res, err := s.DB.ExecContext(ctx, `
		UPDATE chat_topics SET pid = ?, updated_at = ? WHERE id = ?
	`, pidVal, now, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// CloseTopic sets status='closed' and clears the pid.
func (s *Store) CloseTopic(ctx context.Context, id string) error {
	now := time.Now().UnixMilli()
	res, err := s.DB.ExecContext(ctx, `
		UPDATE chat_topics SET status = 'closed', pid = NULL, updated_at = ? WHERE id = ?
	`, now, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ListOpenTopics returns all topics with status='open'. Used for startup
// recovery.
func (s *Store) ListOpenTopics(ctx context.Context) ([]ChatTopic, error) {
	rows, err := s.DB.QueryContext(ctx, topicSelect+` WHERE status = 'open'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ChatTopic{}
	for rows.Next() {
		t, err := scanTopic(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

// CreateMessage inserts a new chat message.
func (s *Store) CreateMessage(ctx context.Context, id string, in MessageCreate) (*ChatMessage, error) {
	now := time.Now().UnixMilli()
	var replyTo any
	if in.InReplyTo != nil {
		replyTo = *in.InReplyTo
	}
	if _, err := s.DB.ExecContext(ctx, `
		INSERT INTO chat_messages (id, topic_id, role, content, in_reply_to, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, id, in.TopicID, in.Role, in.Content, replyTo, now); err != nil {
		return nil, err
	}
	msg := &ChatMessage{
		ID: id, TopicID: in.TopicID, Role: in.Role,
		Content: in.Content, InReplyTo: in.InReplyTo,
		CreatedAt: now,
	}
	return msg, nil
}

// ListMessages returns messages for a topic with cursor-based pagination.
// Messages with created_at > after are returned, up to limit.
func (s *Store) ListMessages(ctx context.Context, topicID string, after int64, limit int) ([]ChatMessage, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, topic_id, role, content, in_reply_to, created_at
		FROM chat_messages
		WHERE topic_id = ? AND created_at > ?
		ORDER BY created_at
		LIMIT ?
	`, topicID, after, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ChatMessage{}
	for rows.Next() {
		var (
			m       ChatMessage
			replyTo sql.NullString
		)
		if err := rows.Scan(&m.ID, &m.TopicID, &m.Role, &m.Content, &replyTo, &m.CreatedAt); err != nil {
			return nil, err
		}
		if replyTo.Valid {
			v := replyTo.String
			m.InReplyTo = &v
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ---- helpers ---------------------------------------------------------------

const topicSelect = `
SELECT id, project_id, title, status, plugin, pid, created_at, updated_at
FROM chat_topics`

func scanTopic(scan func(...any) error) (*ChatTopic, error) {
	var (
		t   ChatTopic
		pid sql.NullInt64
	)
	if err := scan(
		&t.ID, &t.ProjectID, &t.Title, &t.Status, &t.Plugin, &pid,
		&t.CreatedAt, &t.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if pid.Valid {
		v := int(pid.Int64)
		t.PID = &v
	}
	return &t, nil
}
