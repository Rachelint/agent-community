package plugin

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// DataDirs locates the on-disk agent definition directories. The list
// order defines override precedence (earlier wins on duplicate names).
type DataDirs struct {
	// RepoAgentsDir is the agents/ directory bundled with this repo.
	RepoAgentsDir string
	// UserAgentsDir is ~/.agent-community/agents/, used for user overrides.
	UserAgentsDir string
}

func (d DataDirs) list() []string {
	// User overrides win over bundled defaults.
	return []string{d.UserAgentsDir, d.RepoAgentsDir}
}

// Manager owns the spawn lifecycle of agent processes. It is deliberately
// thin: once a process is launched successfully, the manager keeps only
// a weak reference (used for probing). Completion is driven entirely by
// callbacks from the spawned process.
type Manager struct {
	dirs        DataDirs
	callbackURL string // e.g. http://127.0.0.1:8080
	token       string // shared secret handed to plugins for callbacks

	mu   sync.Mutex
	runs map[string]*runtimeRun // key: run_id
}

// runtimeRun is the manager's in-memory view of an active process.
type runtimeRun struct {
	RunID string
	PID   int
	Cmd   *exec.Cmd
}

// NewManager constructs a Manager. callbackURL is the base URL that
// spawned processes should hit (e.g. "http://127.0.0.1:8080"). A fresh
// random token is generated per server start and embedded into each
// agent's environment so callbacks on /plugin can be authenticated.
func NewManager(dirs DataDirs, callbackURL string) (*Manager, error) {
	tk, err := randomToken()
	if err != nil {
		return nil, err
	}
	return &Manager{
		dirs:        dirs,
		callbackURL: callbackURL,
		token:       tk,
		runs:        make(map[string]*runtimeRun),
	}, nil
}

// Token returns the shared secret. Kept private to the host; the only
// legitimate consumer is an agent process launched by this manager.
func (m *Manager) Token() string { return m.token }

// ScanManifests returns the full list of manifests visible to this
// manager.
func (m *Manager) ScanManifests() ([]*Manifest, error) {
	return ScanDirs(m.dirs.list())
}

// LoadManifestByName returns the manifest for a given agent name, or
// nil if not found.
func (m *Manager) LoadManifestByName(name string) (*Manifest, error) {
	all, err := m.ScanManifests()
	if err != nil {
		return nil, err
	}
	for _, mf := range all {
		if mf.Name == name {
			return mf, nil
		}
	}
	return nil, fmt.Errorf("agent %q not found", name)
}

// SpawnArgs carries everything needed to kick off a worker run.
type SpawnArgs struct {
	Manifest          *Manifest
	RunID             string
	IssueWorkspaceDir string // per-issue workspace (contains repo/ and runs/)
	RunDir            string // per-run directory inside issue workspace
	BrainDir          string // project repo_local, may be empty
	Prompt            string // sent on stdin as a worker.start params.prompt
	Extra             map[string]string
}

// Spawn launches the agent process.
//
//   - The process's stdout is line-split and unmarshaled as JSON-RPC
//     notifications. Only worker.ready is handled inline; everything
//     else is forwarded to the callback endpoints (the agent is
//     expected to call /plugin/runs/:id/{log,complete,fail} directly
//     via HTTP rather than rely on stdout, but we still capture stdout
//     as a safety net and for debugging).
//   - The process's stderr is copied to <workspace>/logs/stderr.log.
//
// Spawn waits up to manifest.Timeout() (capped at 10s here) for the
// first ready notification, after which it returns with the pid. The
// caller is responsible for flipping the DB row into 'running'.
//
// readyCtx bounds only how long Spawn is willing to wait for
// worker.ready. It is *not* tied to the child process lifetime —
// cancelling readyCtx after Spawn returns will not kill the worker.
// This is the actor model: once we have a pid, the worker runs on its
// own until it calls back with completion or the user explicitly
// cancels.
func (m *Manager) Spawn(readyCtx context.Context, a SpawnArgs) (pid int, err error) {
	if a.Manifest == nil {
		return 0, errors.New("manifest required")
	}
	if a.RunDir == "" {
		return 0, errors.New("run dir required")
	}
	if err := os.MkdirAll(a.RunDir, 0o755); err != nil {
		return 0, err
	}

	vars := map[string]string{
		"run.id":          a.RunID,
		"run.workspace":   a.IssueWorkspaceDir, // backwards compat alias
		"run.dir":         a.RunDir,
		"issue.workspace": a.IssueWorkspaceDir,
		"brain.dir":       a.BrainDir,
		"topic.dir":       a.IssueWorkspaceDir, // aliased for chat agents later
		"agent.dir":       filepath.Dir(a.Manifest.Path()),
	}
	for k, v := range a.Extra {
		vars[k] = v
	}

	cwd, err := ExpandPlaceholders(a.Manifest.CWD, vars)
	if err != nil {
		return 0, fmt.Errorf("expand cwd: %w", err)
	}
	if cwd == "" {
		cwd = a.IssueWorkspaceDir
	}

	args := make([]string, 0, len(a.Manifest.Args))
	for _, raw := range a.Manifest.Args {
		s, err := ExpandPlaceholders(raw, vars)
		if err != nil {
			return 0, fmt.Errorf("expand arg %q: %w", raw, err)
		}
		args = append(args, s)
	}

	cmd := exec.Command(a.Manifest.Command, args...)
	cmd.Dir = cwd
	cmd.Env = buildEnv(a, m.callbackURL, m.token)

	// Stdin: we write the initial worker.start notification.
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return 0, err
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return 0, err
	}
	// stderr straight to file; we never parse it.
	stderrFile, err := os.Create(filepath.Join(a.RunDir, "stderr.log"))
	if err != nil {
		return 0, err
	}
	cmd.Stderr = stderrFile

	if err := cmd.Start(); err != nil {
		_ = stderrFile.Close()
		return 0, err
	}

	// Send worker.start on stdin, then close it.
	go func() {
		defer stdin.Close()
		payload := map[string]any{
			"jsonrpc": "2.0",
			"method":  "worker.start",
			"params": map[string]any{
				"run_id":       a.RunID,
				"workspace":    a.IssueWorkspaceDir,
				"run_dir":      a.RunDir,
				"brain_dir":    a.BrainDir,
				"prompt":       a.Prompt,
				"callback_url": m.callbackURL,
				"token":        m.token,
			},
		}
		_ = json.NewEncoder(stdin).Encode(payload)
	}()

	// Open stdout log file for tee-ing.
	stdoutFile, err := os.Create(filepath.Join(a.RunDir, "stdout.log"))
	if err != nil {
		_ = cmd.Process.Kill()
		_ = stderrFile.Close()
		return 0, err
	}

	readyCh := make(chan struct{}, 1)
	failCh := make(chan error, 1)

	// Reader goroutine: tee stdout to file AND scan for worker.ready.
	go func() {
		defer stdoutFile.Close()
		// Use TeeReader so every byte read by the scanner is also
		// written to the log file.
		tee := io.TeeReader(stdoutPipe, stdoutFile)
		scanner := bufio.NewScanner(tee)
		scanner.Buffer(make([]byte, 1<<16), 1<<22)
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}
			slog.Debug("plugin stdout", "run_id", a.RunID, "line", string(line))
			var msg struct {
				Method string `json:"method"`
			}
			if err := json.Unmarshal(line, &msg); err != nil {
				continue
			}
			if msg.Method == "worker.ready" {
				select {
				case readyCh <- struct{}{}:
				default:
				}
			}
		}
	}()

	// Watcher goroutine: only used to detect early exit before ready.
	// After Spawn returns, the process runs independently — the
	// reconciler detects completion via done.json.
	go func() {
		err := cmd.Wait()
		_ = stderrFile.Close()
		slog.Debug("plugin process exited", "run_id", a.RunID, "err", err)
		if err != nil {
			failCh <- fmt.Errorf("process exited: %w", err)
		} else {
			failCh <- nil
		}
	}()

	// Ready window — 10s max, but allow manifest to shorten it via
	// timeout_ms (we pick the smaller of the two).
	maxReady := 10 * time.Second
	if t := a.Manifest.Timeout(); t < maxReady && t > 0 {
		maxReady = t
	}
	select {
	case <-readyCtx.Done():
		// Caller lost patience (usually because the HTTP request's
		// context was cancelled). Kill the still-initializing process
		// — we never returned a pid so nobody can track it.
		_ = cmd.Process.Kill()
		return 0, readyCtx.Err()
	case <-readyCh:
		pid = cmd.Process.Pid
		m.track(a.RunID, pid, cmd)
		return pid, nil
	case err := <-failCh:
		if err == nil {
			return 0, errors.New("agent exited before becoming ready")
		}
		return 0, err
	case <-time.After(maxReady):
		_ = cmd.Process.Kill()
		return 0, fmt.Errorf("agent did not become ready within %s", maxReady)
	}
}

func (m *Manager) track(runID string, pid int, cmd *exec.Cmd) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.runs[runID] = &runtimeRun{RunID: runID, PID: pid, Cmd: cmd}
}

// Forget removes the in-memory process handle for a run that has reached
// a terminal state. It does not signal the process.
func (m *Manager) Forget(runID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.runs, runID)
}

// Probe checks whether the worker process for a run is still alive by
// sending signal 0. Returns alive=false if the pid is unknown, the
// process has already been reaped, or the kernel reports ESRCH.
func (m *Manager) Probe(runID string, pidFallback int) (alive bool, pid int) {
	m.mu.Lock()
	r, tracked := m.runs[runID]
	m.mu.Unlock()

	if tracked {
		pid = r.PID
	} else {
		pid = pidFallback
	}
	if pid == 0 {
		return false, 0
	}
	// signal 0 is a no-op delivery: succeeds if the process exists and
	// is reachable by this uid, fails otherwise.
	if err := syscall.Kill(pid, 0); err != nil {
		return false, pid
	}
	return true, pid
}

// Cancel sends SIGTERM to the worker process. If the process is not
// tracked (perhaps we were restarted), we best-effort target the pid
// recorded in the DB via pidFallback.
func (m *Manager) Cancel(runID string, pidFallback int) error {
	m.mu.Lock()
	r, tracked := m.runs[runID]
	m.mu.Unlock()

	pid := pidFallback
	if tracked {
		pid = r.PID
	}
	if pid == 0 {
		return errors.New("no pid to cancel")
	}
	return syscall.Kill(pid, syscall.SIGTERM)
}

// buildEnv composes the env slice for the child process.
func buildEnv(a SpawnArgs, callbackURL, token string) []string {
	env := os.Environ()
	env = append(env,
		"AC_RUN_ID="+a.RunID,
		"AC_WORKSPACE="+a.IssueWorkspaceDir,
		"AC_RUN_DIR="+a.RunDir,
		"AC_BRAIN_DIR="+a.BrainDir,
		"AC_CALLBACK_URL="+callbackURL,
		"AC_PLUGIN_TOKEN="+token,
	)
	for k, v := range a.Manifest.Env {
		expanded, err := ExpandPlaceholders(v, map[string]string{
			"run.id":          a.RunID,
			"run.workspace":   a.IssueWorkspaceDir,
			"run.dir":         a.RunDir,
			"issue.workspace": a.IssueWorkspaceDir,
			"brain.dir":       a.BrainDir,
		})
		if err != nil {
			// Fall back to literal value; spawn validation already ran
			// against the visible placeholders so this shouldn't trip.
			expanded = v
		}
		env = append(env, k+"="+expanded)
	}
	return env
}

func randomToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
