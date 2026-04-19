package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// Project is the canonical representation of a project row.
type Project struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	RepoURL       string `json:"repo_url,omitempty"`
	RepoLocal     string `json:"repo_local"`
	DefaultBranch string `json:"default_branch"`
	CreatedAt     int64  `json:"created_at"`
	UpdatedAt     int64  `json:"updated_at"`
}

// ProjectCreate carries the fields required to insert a new project.
// ID and timestamps are filled in by the store.
type ProjectCreate struct {
	Name          string
	RepoURL       string
	RepoLocal     string
	DefaultBranch string
}

// CreateProject inserts a new project with the given id. Caller supplies
// the id (typically UUIDv7) so it can be echoed back to the client.
func (s *Store) CreateProject(ctx context.Context, id string, p ProjectCreate) (*Project, error) {
	now := time.Now().UnixMilli()
	branch := p.DefaultBranch
	if branch == "" {
		branch = "master"
	}
	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO projects (id, name, repo_url, repo_local, default_branch, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, id, p.Name, nullableString(p.RepoURL), p.RepoLocal, branch, now, now)
	if err != nil {
		return nil, err
	}
	return &Project{
		ID:            id,
		Name:          p.Name,
		RepoURL:       p.RepoURL,
		RepoLocal:     p.RepoLocal,
		DefaultBranch: branch,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

// GetProject returns a single project by id, or ErrNotFound.
func (s *Store) GetProject(ctx context.Context, id string) (*Project, error) {
	row := s.DB.QueryRowContext(ctx, `
		SELECT id, name, IFNULL(repo_url, ''), repo_local, default_branch, created_at, updated_at
		FROM projects WHERE id = ?
	`, id)
	var p Project
	if err := row.Scan(&p.ID, &p.Name, &p.RepoURL, &p.RepoLocal, &p.DefaultBranch, &p.CreatedAt, &p.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

// ListProjects returns all projects, newest first.
func (s *Store) ListProjects(ctx context.Context) ([]Project, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, name, IFNULL(repo_url, ''), repo_local, default_branch, created_at, updated_at
		FROM projects ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Name, &p.RepoURL, &p.RepoLocal, &p.DefaultBranch, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// DeleteProject removes a project. Returns ErrNotFound if no such id.
func (s *Store) DeleteProject(ctx context.Context, id string) error {
	res, err := s.DB.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, id)
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

// nullableString returns a *string that stores NULL when s is empty.
// Used for columns that the schema allows NULL but the Go struct uses
// the zero value to mean "unset".
func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
