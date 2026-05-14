package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Issue is a project-scoped issue row. Labels, child count, and comment
// count are attached when loaded via list/get helpers so the frontend
// can render without a second round trip.
type Issue struct {
	ID            string  `json:"id"`
	ProjectID     string  `json:"project_id"`
	Number        int     `json:"number"`
	ParentID      *string `json:"parent_id,omitempty"`
	Title         string  `json:"title"`
	Body          string  `json:"body"`
	Status        string  `json:"status"` // "open" | "closed"
	Assignee      *string `json:"assignee,omitempty"`
	CreatedAt     int64   `json:"created_at"`
	UpdatedAt     int64   `json:"updated_at"`
	ClosedAt      *int64  `json:"closed_at,omitempty"`
	SourceTopicID *string `json:"source_topic_id,omitempty"`
	Labels        []Label `json:"labels"`
	ChildCount    int     `json:"child_count"`
}

// IssueCreate carries fields for inserting a new issue. ParentID is
// validated by the caller (must be a top-level issue in the same
// project).
type IssueCreate struct {
	Title         string
	Body          string
	ParentID      *string
	Assignee      *string
	LabelIDs      []string
	SourceTopicID *string
}

// CreateIssue inserts an issue, allocating a new per-project number.
// Number assignment and INSERT run in a single transaction to avoid
// races between concurrent creators.
func (s *Store) CreateIssue(
	ctx context.Context,
	id, projectID string,
	in IssueCreate,
) (*Issue, error) {
	now := time.Now().UnixMilli()

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	// Ensure the counter row exists, then bump it.
	if _, err := tx.ExecContext(ctx,
		`INSERT OR IGNORE INTO project_issue_counter (project_id, last_number) VALUES (?, 0)`,
		projectID,
	); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE project_issue_counter SET last_number = last_number + 1 WHERE project_id = ?`,
		projectID,
	); err != nil {
		return nil, err
	}
	var number int
	if err := tx.QueryRowContext(ctx,
		`SELECT last_number FROM project_issue_counter WHERE project_id = ?`, projectID,
	).Scan(&number); err != nil {
		return nil, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO issues (id, project_id, number, parent_id, title, body, status, assignee, source_topic_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 'open', ?, ?, ?, ?)
	`, id, projectID, number, nullableStringPtr(in.ParentID),
		in.Title, in.Body, nullableStringPtr(in.Assignee), nullableStringPtr(in.SourceTopicID), now, now,
	); err != nil {
		return nil, err
	}

	if err := setIssueLabelsTx(ctx, tx, id, in.LabelIDs); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return s.GetIssue(ctx, id)
}

// setIssueLabelsTx replaces the label set for an issue. Caller holds tx.
func setIssueLabelsTx(ctx context.Context, tx *sql.Tx, issueID string, labelIDs []string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM issue_labels WHERE issue_id = ?`, issueID); err != nil {
		return err
	}
	for _, lid := range labelIDs {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO issue_labels (issue_id, label_id) VALUES (?, ?)`, issueID, lid,
		); err != nil {
			return err
		}
	}
	return nil
}

// GetIssue fetches a non-deleted issue with its labels and child count.
func (s *Store) GetIssue(ctx context.Context, id string) (*Issue, error) {
	row := s.DB.QueryRowContext(ctx, `
		SELECT id, project_id, number, parent_id, title, body, status, assignee,
		       source_topic_id, created_at, updated_at, closed_at
		FROM issues
		WHERE id = ? AND deleted_at IS NULL
	`, id)
	var (
		iss           Issue
		parent        sql.NullString
		assignee      sql.NullString
		sourceTopicID sql.NullString
		closedAt      sql.NullInt64
	)
	if err := row.Scan(
		&iss.ID, &iss.ProjectID, &iss.Number, &parent, &iss.Title, &iss.Body,
		&iss.Status, &assignee, &sourceTopicID, &iss.CreatedAt, &iss.UpdatedAt, &closedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	iss.ParentID = nullStringToPtr(parent)
	iss.Assignee = nullStringToPtr(assignee)
	iss.SourceTopicID = nullStringToPtr(sourceTopicID)
	iss.ClosedAt = nullInt64ToPtr(closedAt)
	iss.Labels = []Label{}

	labels, err := s.issueLabels(ctx, id)
	if err != nil {
		return nil, err
	}
	iss.Labels = labels

	if err := s.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM issues WHERE parent_id = ? AND deleted_at IS NULL`, id,
	).Scan(&iss.ChildCount); err != nil {
		return nil, err
	}
	return &iss, nil
}

func (s *Store) issueLabels(ctx context.Context, issueID string) ([]Label, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT l.id, l.project_id, l.name, l.color, IFNULL(l.description,''), l.created_at, l.updated_at
		FROM issue_labels il
		JOIN labels l ON l.id = il.label_id
		WHERE il.issue_id = ?
		ORDER BY l.name
	`, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Label{}
	for rows.Next() {
		var l Label
		if err := rows.Scan(&l.ID, &l.ProjectID, &l.Name, &l.Color, &l.Description, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// IssueFilter constrains list queries. Zero-valued fields are ignored.
type IssueFilter struct {
	Status   string // "open" | "closed" | ""
	Assignee string
	LabelID  string
	ParentID string // "" = any, "null" = only top-level, otherwise specific id
	Q        string // LIKE %q% on title/body
}

// ListIssues returns issues in a project matching the filter. Children
// follow the "only top-level" convention unless ParentID overrides.
// Each returned row carries labels and child_count; fetched in bulk to
// avoid N+1 on the labels join.
func (s *Store) ListIssues(ctx context.Context, projectID string, f IssueFilter) ([]Issue, error) {
	var (
		clauses = []string{"i.project_id = ?", "i.deleted_at IS NULL"}
		args    = []any{projectID}
	)
	if f.Status != "" {
		clauses = append(clauses, "i.status = ?")
		args = append(args, f.Status)
	}
	if f.Assignee != "" {
		clauses = append(clauses, "i.assignee = ?")
		args = append(args, f.Assignee)
	}
	switch f.ParentID {
	case "":
		// no filter on parent
	case "null":
		clauses = append(clauses, "i.parent_id IS NULL")
	default:
		clauses = append(clauses, "i.parent_id = ?")
		args = append(args, f.ParentID)
	}
	if f.Q != "" {
		clauses = append(clauses, "(i.title LIKE ? OR i.body LIKE ?)")
		like := "%" + f.Q + "%"
		args = append(args, like, like)
	}

	// Label filter requires a JOIN; we use EXISTS to keep it composable.
	if f.LabelID != "" {
		clauses = append(clauses, `EXISTS (
			SELECT 1 FROM issue_labels il WHERE il.issue_id = i.id AND il.label_id = ?
		)`)
		args = append(args, f.LabelID)
	}

	query := fmt.Sprintf(`
		SELECT i.id, i.project_id, i.number, i.parent_id, i.title, i.body, i.status, i.assignee,
		       i.source_topic_id, i.created_at, i.updated_at, i.closed_at
		FROM issues i
		WHERE %s
		ORDER BY i.number DESC
	`, strings.Join(clauses, " AND "))

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var issues []Issue
	var ids []string
	for rows.Next() {
		var (
			iss           Issue
			parent        sql.NullString
			assignee      sql.NullString
			sourceTopicID sql.NullString
			closedAt      sql.NullInt64
		)
		if err := rows.Scan(
			&iss.ID, &iss.ProjectID, &iss.Number, &parent, &iss.Title, &iss.Body,
			&iss.Status, &assignee, &sourceTopicID, &iss.CreatedAt, &iss.UpdatedAt, &closedAt,
		); err != nil {
			return nil, err
		}
		iss.ParentID = nullStringToPtr(parent)
		iss.Assignee = nullStringToPtr(assignee)
		iss.SourceTopicID = nullStringToPtr(sourceTopicID)
		iss.ClosedAt = nullInt64ToPtr(closedAt)
		iss.Labels = []Label{}
		issues = append(issues, iss)
		ids = append(ids, iss.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(issues) == 0 {
		return issues, nil
	}

	// Bulk-load labels + child counts for all ids.
	if err := s.attachLabels(ctx, issues, ids); err != nil {
		return nil, err
	}
	if err := s.attachChildCounts(ctx, issues, ids); err != nil {
		return nil, err
	}
	return issues, nil
}

func (s *Store) attachLabels(ctx context.Context, issues []Issue, ids []string) error {
	placeholders, args := placeholders(ids)
	rows, err := s.DB.QueryContext(ctx, fmt.Sprintf(`
		SELECT il.issue_id, l.id, l.project_id, l.name, l.color, IFNULL(l.description,''), l.created_at, l.updated_at
		FROM issue_labels il
		JOIN labels l ON l.id = il.label_id
		WHERE il.issue_id IN (%s)
		ORDER BY l.name
	`, placeholders), args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	idx := make(map[string]int, len(issues))
	for i, iss := range issues {
		idx[iss.ID] = i
	}
	for rows.Next() {
		var issueID string
		var l Label
		if err := rows.Scan(&issueID, &l.ID, &l.ProjectID, &l.Name, &l.Color, &l.Description, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return err
		}
		if i, ok := idx[issueID]; ok {
			issues[i].Labels = append(issues[i].Labels, l)
		}
	}
	return rows.Err()
}

func (s *Store) attachChildCounts(ctx context.Context, issues []Issue, ids []string) error {
	placeholders, args := placeholders(ids)
	rows, err := s.DB.QueryContext(ctx, fmt.Sprintf(`
		SELECT parent_id, COUNT(*) FROM issues
		WHERE parent_id IN (%s) AND deleted_at IS NULL
		GROUP BY parent_id
	`, placeholders), args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var pid string
		var c int
		if err := rows.Scan(&pid, &c); err != nil {
			return err
		}
		counts[pid] = c
	}
	for i := range issues {
		issues[i].ChildCount = counts[issues[i].ID]
	}
	return rows.Err()
}

// IssuePatch carries optional fields for a partial update. Nil = no change.
// Labels is a replacement list when non-nil.
type IssuePatch struct {
	Title    *string
	Body     *string
	Status   *string // "open" | "closed"
	Assignee *string // empty string means unset
	ParentID *string // empty string means unset
	Labels   *[]string
}

// UpdateIssue applies a partial update.
func (s *Store) UpdateIssue(ctx context.Context, id string, p IssuePatch) (*Issue, error) {
	// Read the current row *before* opening a transaction. The DB is
	// configured with MaxOpenConns=1, so mixing s.DB reads with an open
	// tx would deadlock on the single connection.
	cur, err := s.GetIssue(ctx, id)
	if err != nil {
		return nil, err
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now().UnixMilli()

	var sets []string
	var args []any
	if p.Title != nil {
		sets = append(sets, "title = ?")
		args = append(args, *p.Title)
	}
	if p.Body != nil {
		sets = append(sets, "body = ?")
		args = append(args, *p.Body)
	}
	if p.Status != nil {
		sets = append(sets, "status = ?")
		args = append(args, *p.Status)
		if *p.Status == "closed" && cur.ClosedAt == nil {
			sets = append(sets, "closed_at = ?")
			args = append(args, now)
		} else if *p.Status == "open" {
			sets = append(sets, "closed_at = NULL")
		}
	}
	if p.Assignee != nil {
		if *p.Assignee == "" {
			sets = append(sets, "assignee = NULL")
		} else {
			sets = append(sets, "assignee = ?")
			args = append(args, *p.Assignee)
		}
	}
	if p.ParentID != nil {
		if *p.ParentID == "" {
			sets = append(sets, "parent_id = NULL")
		} else {
			sets = append(sets, "parent_id = ?")
			args = append(args, *p.ParentID)
		}
	}
	if len(sets) > 0 {
		sets = append(sets, "updated_at = ?")
		args = append(args, now)
		args = append(args, id)
		q := fmt.Sprintf("UPDATE issues SET %s WHERE id = ?", strings.Join(sets, ", "))
		if _, err := tx.ExecContext(ctx, q, args...); err != nil {
			return nil, err
		}
	}
	if p.Labels != nil {
		if err := setIssueLabelsTx(ctx, tx, id, *p.Labels); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetIssue(ctx, id)
}

// SoftDeleteIssue marks an issue as deleted.
func (s *Store) SoftDeleteIssue(ctx context.Context, id string) error {
	now := time.Now().UnixMilli()
	res, err := s.DB.ExecContext(ctx,
		`UPDATE issues SET deleted_at = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`,
		now, now, id,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ListIssuesByTopic returns issues linked to a chat topic.
func (s *Store) ListIssuesByTopic(ctx context.Context, topicID string) ([]Issue, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT i.id, i.project_id, i.number, i.parent_id, i.title, i.body, i.status, i.assignee,
		       i.source_topic_id, i.created_at, i.updated_at, i.closed_at
		FROM issues i
		WHERE i.source_topic_id = ? AND i.deleted_at IS NULL
		ORDER BY i.number DESC
	`, topicID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var issues []Issue
	var ids []string
	for rows.Next() {
		var (
			iss           Issue
			parent        sql.NullString
			assignee      sql.NullString
			sourceTopicID sql.NullString
			closedAt      sql.NullInt64
		)
		if err := rows.Scan(
			&iss.ID, &iss.ProjectID, &iss.Number, &parent, &iss.Title, &iss.Body,
			&iss.Status, &assignee, &sourceTopicID, &iss.CreatedAt, &iss.UpdatedAt, &closedAt,
		); err != nil {
			return nil, err
		}
		iss.ParentID = nullStringToPtr(parent)
		iss.Assignee = nullStringToPtr(assignee)
		iss.SourceTopicID = nullStringToPtr(sourceTopicID)
		iss.ClosedAt = nullInt64ToPtr(closedAt)
		iss.Labels = []Label{}
		issues = append(issues, iss)
		ids = append(ids, iss.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(issues) == 0 {
		return issues, nil
	}
	if err := s.attachLabels(ctx, issues, ids); err != nil {
		return nil, err
	}
	if err := s.attachChildCounts(ctx, issues, ids); err != nil {
		return nil, err
	}
	return issues, nil
}

// ---- tiny helpers ----

func nullableStringPtr(p *string) any {
	if p == nil || *p == "" {
		return nil
	}
	return *p
}

func nullStringToPtr(n sql.NullString) *string {
	if !n.Valid {
		return nil
	}
	v := n.String
	return &v
}

func nullInt64ToPtr(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	v := n.Int64
	return &v
}

func placeholders(ids []string) (string, []any) {
	args := make([]any, len(ids))
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = "?"
		args[i] = id
	}
	return strings.Join(parts, ","), args
}
