package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// Label is a project-scoped tag applied to issues.
type Label struct {
	ID          string `json:"id"`
	ProjectID   string `json:"project_id"`
	Name        string `json:"name"`
	Color       string `json:"color"` // hex without leading "#"
	Description string `json:"description,omitempty"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

// LabelCreate carries fields for inserting a label.
type LabelCreate struct {
	Name        string
	Color       string
	Description string
}

// CreateLabel inserts a label. Returns ErrConflict if (project, name)
// already exists.
func (s *Store) CreateLabel(ctx context.Context, id, projectID string, in LabelCreate) (*Label, error) {
	now := time.Now().UnixMilli()
	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO labels (id, project_id, name, color, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, id, projectID, in.Name, in.Color, nullableString(in.Description), now, now)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, err
	}
	return &Label{
		ID: id, ProjectID: projectID,
		Name: in.Name, Color: in.Color, Description: in.Description,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

// ListLabels returns all labels for a project, ordered by name.
func (s *Store) ListLabels(ctx context.Context, projectID string) ([]Label, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, project_id, name, color, IFNULL(description,''), created_at, updated_at
		FROM labels WHERE project_id = ? ORDER BY name
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Label
	for rows.Next() {
		var l Label
		if err := rows.Scan(&l.ID, &l.ProjectID, &l.Name, &l.Color, &l.Description, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// GetLabel fetches a label by id.
func (s *Store) GetLabel(ctx context.Context, id string) (*Label, error) {
	row := s.DB.QueryRowContext(ctx, `
		SELECT id, project_id, name, color, IFNULL(description,''), created_at, updated_at
		FROM labels WHERE id = ?
	`, id)
	var l Label
	if err := row.Scan(&l.ID, &l.ProjectID, &l.Name, &l.Color, &l.Description, &l.CreatedAt, &l.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &l, nil
}

// LabelPatch carries optional fields for a partial update.
type LabelPatch struct {
	Name        *string
	Color       *string
	Description *string
}

// UpdateLabel applies a partial update. Returns ErrNotFound if missing.
func (s *Store) UpdateLabel(ctx context.Context, id string, p LabelPatch) (*Label, error) {
	cur, err := s.GetLabel(ctx, id)
	if err != nil {
		return nil, err
	}
	if p.Name != nil {
		cur.Name = *p.Name
	}
	if p.Color != nil {
		cur.Color = *p.Color
	}
	if p.Description != nil {
		cur.Description = *p.Description
	}
	cur.UpdatedAt = time.Now().UnixMilli()
	_, err = s.DB.ExecContext(ctx, `
		UPDATE labels SET name=?, color=?, description=?, updated_at=? WHERE id=?
	`, cur.Name, cur.Color, nullableString(cur.Description), cur.UpdatedAt, id)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, err
	}
	return cur, nil
}

// DeleteLabel removes a label.
func (s *Store) DeleteLabel(ctx context.Context, id string) error {
	res, err := s.DB.ExecContext(ctx, `DELETE FROM labels WHERE id=?`, id)
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
