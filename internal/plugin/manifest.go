// Package plugin manages chat and worker agent processes.
//
// An agent is an external command described by an agent.json manifest.
// Both chat and worker agents follow the same pattern: the host spawns
// the process, verifies startup, then steps back. The process drives
// progress by writing JSON-RPC notifications to stdout and/or by
// calling back into the host's /plugin/* endpoints.
//
// Phase 3 focuses on worker agents. Chat wiring arrives in phase 5 and
// reuses the same spawn machinery.
package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Manifest mirrors the JSON schema of agent.json files.
type Manifest struct {
	Name           string            `json:"name"`
	Version        string            `json:"version,omitempty"`
	Kind           string            `json:"kind"` // "chat" | "worker"
	Command        string            `json:"command"`
	Args           []string          `json:"args,omitempty"`
	CWD            string            `json:"cwd,omitempty"`
	Env            map[string]string `json:"env,omitempty"`
	Protocol       string            `json:"protocol"` // "jsonrpc-stdio"
	TimeoutMS      int               `json:"timeout_ms,omitempty"`
	Capabilities   []string          `json:"capabilities,omitempty"`
	PromptTemplate string            `json:"prompt_template,omitempty"`

	// path where this manifest was loaded from — populated by the loader,
	// not read from JSON.
	path string `json:"-"`
}

// Path returns the absolute filesystem location the manifest was loaded
// from.
func (m *Manifest) Path() string { return m.path }

// Timeout returns the configured manifest timeout or a default of 10
// minutes.
func (m *Manifest) Timeout() time.Duration {
	if m.TimeoutMS <= 0 {
		return 10 * time.Minute
	}
	return time.Duration(m.TimeoutMS) * time.Millisecond
}

// LoadManifest reads and validates a single agent.json file.
func LoadManifest(path string) (*Manifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest %s: %w", path, err)
	}
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("parse manifest %s: %w", path, err)
	}
	m.path = path
	if err := m.validate(); err != nil {
		return nil, fmt.Errorf("validate manifest %s: %w", path, err)
	}
	return &m, nil
}

func (m *Manifest) validate() error {
	if m.Name == "" {
		return errors.New("name is required")
	}
	switch m.Kind {
	case "chat", "worker":
	default:
		return fmt.Errorf("kind must be chat or worker, got %q", m.Kind)
	}
	if m.Command == "" {
		return errors.New("command is required")
	}
	if m.Protocol != "jsonrpc-stdio" {
		return fmt.Errorf("unsupported protocol %q (only jsonrpc-stdio)", m.Protocol)
	}
	return nil
}

// ScanDirs walks every path in dirs and loads agent.json files matching
// <dir>/<name>/agent.json. Missing directories are ignored. Returns
// manifests in stable order (sorted by name).
func ScanDirs(dirs []string) ([]*Manifest, error) {
	seen := make(map[string]*Manifest)
	var order []string
	for _, d := range dirs {
		if d == "" {
			continue
		}
		info, err := os.Stat(d)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, err
		}
		if !info.IsDir() {
			continue
		}
		entries, err := os.ReadDir(d)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			mf := filepath.Join(d, e.Name(), "agent.json")
			if _, err := os.Stat(mf); err != nil {
				continue
			}
			m, err := LoadManifest(mf)
			if err != nil {
				return nil, err
			}
			if _, dup := seen[m.Name]; dup {
				// Earlier dirs win — e.g. a user override in
				// ~/.agent-community/agents can be picked up first
				// by listing it earlier in dirs.
				continue
			}
			seen[m.Name] = m
			order = append(order, m.Name)
		}
	}
	out := make([]*Manifest, 0, len(order))
	for _, n := range order {
		out = append(out, seen[n])
	}
	return out, nil
}

// ExpandPlaceholders substitutes ${key} tokens in s using vars. Unknown
// keys produce an error so typos in manifests are caught at spawn time
// rather than silently resolved to an empty string.
func ExpandPlaceholders(s string, vars map[string]string) (string, error) {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if strings.HasPrefix(s[i:], "${") {
			end := strings.Index(s[i:], "}")
			if end == -1 {
				return "", fmt.Errorf("unterminated placeholder in %q", s)
			}
			key := s[i+2 : i+end]
			v, ok := vars[key]
			if !ok {
				return "", fmt.Errorf("unknown placeholder ${%s}", key)
			}
			b.WriteString(v)
			i += end + 1
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String(), nil
}
