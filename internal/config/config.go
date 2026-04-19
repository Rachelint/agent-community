// Package config loads runtime configuration from environment variables.
// Later phases will add a config file layer.
package config

import (
	"os"
	"path/filepath"
)

// Config holds server runtime configuration.
type Config struct {
	// Addr is the listen address, e.g. ":8080".
	Addr string
	// DataDir is the root where the database and workspaces live.
	// Defaults to ~/.agent-community.
	DataDir string
	// RepoAgentsDir points at the agents/ directory bundled with this
	// repository — used as the default manifest source.
	RepoAgentsDir string
	// UserAgentsDir is DataDir/agents. Manifests here override the
	// repo-bundled ones.
	UserAgentsDir string
	// CallbackBaseURL is the URL prefix spawned agents use to reach
	// the host (defaults to http://127.0.0.1<Addr>).
	CallbackBaseURL string
}

// Load reads configuration from environment variables, applying sensible
// defaults. It never fails: callers get a usable Config regardless.
func Load() Config {
	addr := os.Getenv("AC_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	dataDir := os.Getenv("AC_DATA_DIR")
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			dataDir = filepath.Join(home, ".agent-community")
		}
	}

	repoAgents := os.Getenv("AC_REPO_AGENTS_DIR")
	if repoAgents == "" {
		// Default: <cwd>/agents. This works in `make dev`, `go run`,
		// and the shipped binary (assuming the binary is started from
		// the repo root). Can be overridden with AC_REPO_AGENTS_DIR.
		if wd, err := os.Getwd(); err == nil {
			repoAgents = filepath.Join(wd, "agents")
		}
	}

	callback := os.Getenv("AC_CALLBACK_URL")
	if callback == "" {
		// The child process lives on the same host, so we hard-code
		// 127.0.0.1 — binding to a specific interface via AC_ADDR
		// shouldn't change where the callback goes.
		callback = "http://127.0.0.1" + addr
	}

	return Config{
		Addr:            addr,
		DataDir:         dataDir,
		RepoAgentsDir:   repoAgents,
		UserAgentsDir:   filepath.Join(dataDir, "agents"),
		CallbackBaseURL: callback,
	}
}
