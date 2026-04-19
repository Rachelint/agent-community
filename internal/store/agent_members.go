package store

import (
	"context"
	"database/sql"
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
