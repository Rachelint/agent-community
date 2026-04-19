package store

import (
	"context"
	"database/sql"
	"time"
)

// Notification is a mailbox entry. Phase 3 only writes them (on
// worker_complete/fail); phase 4 adds the read/archive UI.
type Notification struct {
	ID         string  `json:"id"`
	ProjectID  string  `json:"project_id"`
	Kind       string  `json:"kind"`
	IssueID    *string `json:"issue_id,omitempty"`
	RunID      *string `json:"run_id,omitempty"`
	Title      string  `json:"title"`
	Body       string  `json:"body,omitempty"`
	ReadAt     *int64  `json:"read_at,omitempty"`
	ArchivedAt *int64  `json:"archived_at,omitempty"`
	CreatedAt  int64   `json:"created_at"`
}

// NotificationCreate is the input for CreateNotification.
type NotificationCreate struct {
	ProjectID string
	Kind      string
	IssueID   string
	RunID     string
	Title     string
	Body      string
}

// CreateNotification inserts an unread, unarchived notification.
func (s *Store) CreateNotification(ctx context.Context, id string, in NotificationCreate) (*Notification, error) {
	now := time.Now().UnixMilli()
	if _, err := s.DB.ExecContext(ctx, `
		INSERT INTO notifications (id, project_id, kind, issue_id, run_id, title, body, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, id, in.ProjectID, in.Kind,
		nullableString(in.IssueID), nullableString(in.RunID),
		in.Title, nullableString(in.Body), now,
	); err != nil {
		return nil, err
	}
	n := &Notification{
		ID: id, ProjectID: in.ProjectID, Kind: in.Kind,
		Title: in.Title, Body: in.Body, CreatedAt: now,
	}
	if in.IssueID != "" {
		n.IssueID = &in.IssueID
	}
	if in.RunID != "" {
		n.RunID = &in.RunID
	}
	return n, nil
}

// ListNotifications returns notifications for a project, newest first.
// If unreadOnly is true, read_at IS NULL AND archived_at IS NULL.
func (s *Store) ListNotifications(ctx context.Context, projectID string, unreadOnly bool) ([]Notification, error) {
	q := `
		SELECT id, project_id, kind, issue_id, run_id, title, IFNULL(body,''),
		       read_at, archived_at, created_at
		FROM notifications
		WHERE project_id = ?`
	if unreadOnly {
		q += ` AND read_at IS NULL AND archived_at IS NULL`
	}
	q += ` ORDER BY created_at DESC`
	rows, err := s.DB.QueryContext(ctx, q, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Notification{}
	for rows.Next() {
		var (
			n        Notification
			issueID  sql.NullString
			runID    sql.NullString
			readAt   sql.NullInt64
			archivedAt sql.NullInt64
		)
		if err := rows.Scan(
			&n.ID, &n.ProjectID, &n.Kind, &issueID, &runID, &n.Title, &n.Body,
			&readAt, &archivedAt, &n.CreatedAt,
		); err != nil {
			return nil, err
		}
		if issueID.Valid {
			v := issueID.String
			n.IssueID = &v
		}
		if runID.Valid {
			v := runID.String
			n.RunID = &v
		}
		if readAt.Valid {
			v := readAt.Int64
			n.ReadAt = &v
		}
		if archivedAt.Valid {
			v := archivedAt.Int64
			n.ArchivedAt = &v
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// MarkNotificationRead sets read_at on a notification. Idempotent.
func (s *Store) MarkNotificationRead(ctx context.Context, id string) error {
	now := time.Now().UnixMilli()
	res, err := s.DB.ExecContext(ctx, `
		UPDATE notifications SET read_at = ? WHERE id = ? AND read_at IS NULL
	`, now, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		// Check if it exists at all.
		var exists int
		if err := s.DB.QueryRowContext(ctx, `SELECT 1 FROM notifications WHERE id = ?`, id).Scan(&exists); err != nil {
			return ErrNotFound
		}
	}
	return nil
}

// ArchiveNotification sets archived_at on a notification. Idempotent.
func (s *Store) ArchiveNotification(ctx context.Context, id string) error {
	now := time.Now().UnixMilli()
	res, err := s.DB.ExecContext(ctx, `
		UPDATE notifications SET archived_at = ? WHERE id = ? AND archived_at IS NULL
	`, now, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		var exists int
		if err := s.DB.QueryRowContext(ctx, `SELECT 1 FROM notifications WHERE id = ?`, id).Scan(&exists); err != nil {
			return ErrNotFound
		}
	}
	return nil
}

// CountUnreadNotifications returns the number of unread, unarchived
// notifications for a project.
func (s *Store) CountUnreadNotifications(ctx context.Context, projectID string) (int, error) {
	var count int
	err := s.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM notifications
		WHERE project_id = ? AND read_at IS NULL AND archived_at IS NULL
	`, projectID).Scan(&count)
	return count, err
}
