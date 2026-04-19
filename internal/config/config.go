// Package config loads runtime configuration from environment variables.
// Later phases will add a config file layer.
package config

import "os"

// Config holds server runtime configuration.
type Config struct {
	// Addr is the listen address, e.g. ":8080".
	Addr string
	// DataDir is the root where the database and workspaces live.
	// Defaults to ~/.agent-community.
	DataDir string
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
			dataDir = home + "/.agent-community"
		}
	}
	return Config{
		Addr:    addr,
		DataDir: dataDir,
	}
}
