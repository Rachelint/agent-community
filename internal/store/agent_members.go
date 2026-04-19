package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// AgentMember is a registered chat or worker agent.
type AgentMember struct {
	Name         string `json:"name"`
	Kind         string `json:"kind"` // "chat" | "worker"
	ManifestPath string `json:"manifest_path"`
	Enabled      bool   `json:"enabled"`
	RegisteredAt int64  `json:"registered_at"`
	UpdatedAt    int64  `json:"updated_at"`
}

// ListAgentMembers returns all registered agents ordered by kind then name.
// When kind is non-empty, only members of that kind are returned.
func (s *Store) ListAgentMembers(ctx context.Context, kind string) ([]AgentMember, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if kind != "" {
		rows, err = s.DB.QueryContext(ctx, `
			SELECT name, kind, manifest_path, enabled, registered_at, updated_at
			FROM agent_members WHERE kind = ? ORDER BY name
		`, kind)
	} else {
		rows, err = s.DB.QueryContext(ctx, `
			SELECT name, kind, manifest_path, enabled, registered_at, updated_at
			FROM agent_members ORDER BY kind, name
		`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AgentMember
	for rows.Next() {
		var m AgentMember
		var enabled int
		if err := rows.Scan(&m.Name, &m.Kind, &m.ManifestPath, &enabled, &m.RegisteredAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		m.Enabled = enabled != 0
		out = append(out, m)
	}
	return out, rows.Err()
}

// GetAgentMember fetches a single registration.
func (s *Store) GetAgentMember(ctx context.Context, name string) (*AgentMember, error) {
	row := s.DB.QueryRowContext(ctx, `
		SELECT name, kind, manifest_path, enabled, registered_at, updated_at
		FROM agent_members WHERE name = ?
	`, name)
	var m AgentMember
	var enabled int
	if err := row.Scan(&m.Name, &m.Kind, &m.ManifestPath, &enabled, &m.RegisteredAt, &m.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	m.Enabled = enabled != 0
	return &m, nil
}

// UpsertAgentMember inserts or updates a registration. enabled defaults
// to true on first insert; re-scans that find a known name refresh
// kind/manifest_path but preserve the existing enabled flag (the user's
// choice wins over manifest rescans).
func (s *Store) UpsertAgentMember(ctx context.Context, name, kind, manifestPath string) error {
	now := time.Now().UnixMilli()
	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO agent_members (name, kind, manifest_path, enabled, registered_at, updated_at)
		VALUES (?, ?, ?, 1, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			kind          = excluded.kind,
			manifest_path = excluded.manifest_path,
			updated_at    = excluded.updated_at
	`, name, kind, manifestPath, now, now)
	return err
}

// DisableAgentMember flips the enabled flag off. Used by reload when a
// manifest disappears from disk — we don't drop the row because dispatch
// history and pending runs may still reference the name.
func (s *Store) DisableAgentMember(ctx context.Context, name string) error {
	now := time.Now().UnixMilli()
	res, err := s.DB.ExecContext(ctx, `
		UPDATE agent_members SET enabled = 0, updated_at = ? WHERE name = ?
	`, now, name)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
