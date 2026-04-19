// Package store owns database access: migrations, sqlc-generated queries,
// and a thin Open/Close API over modernc.org/sqlite.
package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite" // pure-Go SQLite driver
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Store wraps a *sql.DB handle with agent-community conventions:
// WAL mode, foreign keys on, busy timeout.
type Store struct {
	DB *sql.DB
}

// Open initializes a SQLite database at dbPath, creating parent dirs if
// needed, and runs any pending migrations.
func Open(ctx context.Context, dbPath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	// _pragma params are applied on every connection. We use the
	// modernc.org/sqlite DSN syntax; pragmas that take effect per-conn
	// (foreign_keys, busy_timeout) must be set this way.
	dsn := fmt.Sprintf(
		"file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(on)&_pragma=busy_timeout(5000)",
		dbPath,
	)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// SQLite write concurrency is limited; a single connection keeps
	// migrations and transactions serialized, avoiding "database is locked".
	db.SetMaxOpenConns(1)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	if err := migrate(ctx, db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return &Store{DB: db}, nil
}

// Close releases the underlying database handle.
func (s *Store) Close() error { return s.DB.Close() }

// migrate applies all embedded migrations whose version is greater than
// the current schema version. Migrations are plain .sql files named like
// 0001_init.sql under db/migrations/. Each file runs inside its own
// transaction and the version is recorded in schema_migrations.
func migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version     INTEGER PRIMARY KEY,
			name        TEXT NOT NULL,
			applied_at  INTEGER NOT NULL
		);
	`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	type migration struct {
		version int
		name    string
		path    string
	}
	var all []migration
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		// Expect prefix "NNNN_" (four digits + underscore).
		if len(e.Name()) < 5 || e.Name()[4] != '_' {
			return fmt.Errorf("migration %q must be named NNNN_*.sql", e.Name())
		}
		var v int
		if _, err := fmt.Sscanf(e.Name()[:4], "%d", &v); err != nil {
			return fmt.Errorf("migration %q: parse version: %w", e.Name(), err)
		}
		all = append(all, migration{
			version: v,
			name:    strings.TrimSuffix(e.Name(), ".sql"),
			path:    "migrations/" + e.Name(),
		})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].version < all[j].version })

	// Find current version.
	var current int
	if err := db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&current); err != nil {
		return fmt.Errorf("read current version: %w", err)
	}

	for _, m := range all {
		if m.version <= current {
			continue
		}
		if m.version != current+1 {
			return fmt.Errorf("migration gap: have %d, next is %d", current, m.version)
		}
		content, err := migrationsFS.ReadFile(m.path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", m.name, err)
		}
		if err := applyMigration(ctx, db, m.version, m.name, string(content)); err != nil {
			return fmt.Errorf("apply migration %s: %w", m.name, err)
		}
		current = m.version
	}
	return nil
}

func applyMigration(ctx context.Context, db *sql.DB, version int, name, sqlText string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, sqlText); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, strftime('%s','now') * 1000)`,
		version, name,
	); err != nil {
		return err
	}
	return tx.Commit()
}

// ErrNotFound indicates a query expected at least one row but got none.
var ErrNotFound = errors.New("not found")
