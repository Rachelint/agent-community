package store

import (
	"context"
	"time"
)

// IssueComment is a single comment attached to an issue.
type IssueComment struct {
	ID        string `json:"id"`
	IssueID   string `json:"issue_id"`
	Author    string `json:"author"`
	Body      string `json:"body"`
	CreatedAt int64  `json:"created_at"`
}

// CreateIssueComment appends a comment.
func (s *Store) CreateIssueComment(ctx context.Context, id, issueID, author, body string) (*IssueComment, error) {
	now := time.Now().UnixMilli()
	if _, err := s.DB.ExecContext(ctx, `
		INSERT INTO issue_comments (id, issue_id, author, body, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, id, issueID, author, body, now); err != nil {
		return nil, err
	}
	return &IssueComment{
		ID: id, IssueID: issueID, Author: author, Body: body, CreatedAt: now,
	}, nil
}

// ListIssueComments returns comments in chronological order.
func (s *Store) ListIssueComments(ctx context.Context, issueID string) ([]IssueComment, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, issue_id, author, body, created_at
		FROM issue_comments WHERE issue_id = ? ORDER BY created_at
	`, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []IssueComment{}
	for rows.Next() {
		var c IssueComment
		if err := rows.Scan(&c.ID, &c.IssueID, &c.Author, &c.Body, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
