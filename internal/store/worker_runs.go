package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// Run statuses. Keep in sync with the CHECK constraint in migration
// 0003.
const (
	RunQueued      = "queued"
	RunRunning     = "running"
	RunNeedsReview = "needs_review"
	RunCompleted   = "completed"
	RunFailed      = "failed"
	RunCancelled   = "cancelled"
	RunOrphan      = "orphan"
)

// WorkerRun represents one dispatch attempt of a worker agent against
// an issue. Runs are append-only history; the UI surfaces the most
// recent ones inside the issue view.
type WorkerRun struct {
	ID           string `json:"id"`
	IssueID      string `json:"issue_id"`
	ProjectID    string `json:"project_id"`
	Plugin       string `json:"plugin"`
	Status       string `json:"status"`
	PID          *int   `json:"pid,omitempty"`
	WorkspaceDir string `json:"workspace_dir"`
	StartedAt    *int64 `json:"started_at,omitempty"`
	FinishedAt   *int64 `json:"finished_at,omitempty"`
	ExitCode     *int   `json:"exit_code,omitempty"`
	MRURL        string `json:"mr_url,omitempty"`
	Summary      string `json:"summary,omitempty"`
	CreatedAt    int64  `json:"created_at"`
}

// RunCreate holds the fields needed at insert time. A just-created run
// always starts in 'queued' — the worker lifecycle flips it to
// 'running' once the process spawns successfully.
type RunCreate struct {
	IssueID      string
	ProjectID    string
	Plugin       string
	WorkspaceDir string
}

// CreateRun inserts a new queued run. The id is caller-supplied.
func (s *Store) CreateRun(ctx context.Context, id string, in RunCreate) (*WorkerRun, error) {
	now := time.Now().UnixMilli()
	if _, err := s.DB.ExecContext(ctx, `
		INSERT INTO worker_runs (id, issue_id, project_id, plugin, status, workspace_dir, created_at)
		VALUES (?, ?, ?, ?, 'queued', ?, ?)
	`, id, in.IssueID, in.ProjectID, in.Plugin, in.WorkspaceDir, now); err != nil {
		return nil, err
	}
	return &WorkerRun{
		ID: id, IssueID: in.IssueID, ProjectID: in.ProjectID,
		Plugin: in.Plugin, Status: RunQueued, WorkspaceDir: in.WorkspaceDir,
		CreatedAt: now,
	}, nil
}

// GetRun fetches a run by id.
func (s *Store) GetRun(ctx context.Context, id string) (*WorkerRun, error) {
	row := s.DB.QueryRowContext(ctx, runSelect+` WHERE id = ?`, id)
	return scanRun(row.Scan)
}

// ListRunsByIssue returns runs for an issue, newest first.
func (s *Store) ListRunsByIssue(ctx context.Context, issueID string) ([]WorkerRun, error) {
	rows, err := s.DB.QueryContext(ctx, runSelect+` WHERE issue_id = ? ORDER BY created_at DESC`, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []WorkerRun{}
	for rows.Next() {
		r, err := scanRun(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

// RunningRunForIssue returns the currently running run for an issue,
// or nil if none. Used by dispatch to enforce one-at-a-time.
func (s *Store) RunningRunForIssue(ctx context.Context, issueID string) (*WorkerRun, error) {
	row := s.DB.QueryRowContext(ctx, runSelect+`
		WHERE issue_id = ? AND status = 'running' LIMIT 1
	`, issueID)
	r, err := scanRun(row.Scan)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	return r, err
}

// MarkRunRunning sets status=running, stamps started_at, and records pid.
// Safe to call only on a queued run; no-op otherwise.
func (s *Store) MarkRunRunning(ctx context.Context, id string, pid int) error {
	now := time.Now().UnixMilli()
	res, err := s.DB.ExecContext(ctx, `
		UPDATE worker_runs
		SET status='running', started_at=?, pid=?
		WHERE id=? AND status='queued'
	`, now, pid, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// FinishRun moves a running run into a terminal status. Caller supplies
// the final status (completed / failed / cancelled). mrURL and summary
// are taken from the worker's complete callback when present.
func (s *Store) FinishRun(
	ctx context.Context,
	id, status, mrURL, summary string,
	exitCode *int,
) error {
	now := time.Now().UnixMilli()
	var ec any
	if exitCode != nil {
		ec = *exitCode
	}
	res, err := s.DB.ExecContext(ctx, `
		UPDATE worker_runs
		SET status=?, finished_at=?, exit_code=?, mr_url=?, summary=?, pid=NULL
		WHERE id=? AND status='running'
	`, status, now, ec, nullableString(mrURL), nullableString(summary), id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// MarkRunOrphan transitions a running run to orphan. Only allowed for
// status=running, so double-clicks from the UI are idempotent.
func (s *Store) MarkRunOrphan(ctx context.Context, id string) error {
	now := time.Now().UnixMilli()
	res, err := s.DB.ExecContext(ctx, `
		UPDATE worker_runs
		SET status='orphan', finished_at=?, pid=NULL
		WHERE id=? AND status='running'
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

// ListRunningRuns returns all runs with status='running'.
func (s *Store) ListRunningRuns(ctx context.Context) ([]WorkerRun, error) {
	rows, err := s.DB.QueryContext(ctx, runSelect+` WHERE status = 'running'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []WorkerRun{}
	for rows.Next() {
		r, err := scanRun(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

// ---- helpers --------------------------------------------------------------

const runSelect = `
SELECT id, issue_id, project_id, plugin, status, pid,
       workspace_dir, started_at, finished_at, exit_code,
       IFNULL(mr_url,''), IFNULL(summary,''), created_at
FROM worker_runs`

// scanRun adapts either *sql.Row.Scan or *sql.Rows.Scan via the common
// Scan signature.
func scanRun(scan func(...any) error) (*WorkerRun, error) {
	var (
		r        WorkerRun
		pid      sql.NullInt64
		started  sql.NullInt64
		finished sql.NullInt64
		exitCode sql.NullInt64
	)
	if err := scan(
		&r.ID, &r.IssueID, &r.ProjectID, &r.Plugin, &r.Status, &pid,
		&r.WorkspaceDir, &started, &finished, &exitCode,
		&r.MRURL, &r.Summary, &r.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if pid.Valid {
		v := int(pid.Int64)
		r.PID = &v
	}
	if started.Valid {
		v := started.Int64
		r.StartedAt = &v
	}
	if finished.Valid {
		v := finished.Int64
		r.FinishedAt = &v
	}
	if exitCode.Valid {
		v := int(exitCode.Int64)
		r.ExitCode = &v
	}
	return &r, nil
}
