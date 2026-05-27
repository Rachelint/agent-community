package plugin

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"

	"github.com/Rachelint/agent-community/internal/store"
)

// ChatSession tracks a running chat agent process.
type ChatSession struct {
	TopicID string
	PID     int
	Cmd     *exec.Cmd
	Stdin   io.WriteCloser
	mu      sync.Mutex // guards writes to stdin
}

// ChatManager owns the lifecycle of chat agent processes.
type ChatManager struct {
	mu       sync.RWMutex
	sessions map[string]*ChatSession // key: topic_id
	store    *store.Store
	dirs     DataDirs
	dataDir  string // e.g. ~/.agent-community, used for topics/<id>/ state dirs
	baseURL  string // e.g. http://127.0.0.1:8080
}

// NewChatManager constructs a ChatManager. dataDir is the application
// data directory (e.g. ~/.agent-community) where per-topic state
// directories are created.
func NewChatManager(st *store.Store, dirs DataDirs, dataDir, baseURL string) *ChatManager {
	return &ChatManager{
		sessions: make(map[string]*ChatSession),
		store:    st,
		dirs:     dirs,
		dataDir:  dataDir,
		baseURL:  baseURL,
	}
}

// Start spawns a chat agent, sends chat.start, waits for worker.ready,
// and starts a background stdout reader goroutine.
func (cm *ChatManager) Start(
	ctx context.Context,
	topicID string,
	manifest *Manifest,
	projectDir string,
	history []store.ChatMessage,
) (pid int, err error) {
	if manifest == nil {
		return 0, errors.New("manifest required")
	}

	// Create per-topic state directory.
	topicStateDir := filepath.Join(cm.dataDir, "topics", topicID)
	if err := os.MkdirAll(topicStateDir, 0o755); err != nil {
		return 0, fmt.Errorf("create topic state dir: %w", err)
	}
	if err := InitProjectContext(projectDir, cm.dirs); err != nil {
		return 0, fmt.Errorf("init project context: %w", err)
	}

	vars := map[string]string{
		"topic.dir": topicStateDir,
		"brain.dir": projectDir,
		"agent.dir": filepath.Dir(manifest.Path()),
	}

	cwd, err := ExpandPlaceholders(manifest.CWD, vars)
	if err != nil {
		return 0, fmt.Errorf("expand cwd: %w", err)
	}
	if cwd == "" {
		cwd = projectDir
	}

	args := make([]string, 0, len(manifest.Args))
	for _, raw := range manifest.Args {
		s, err := ExpandPlaceholders(raw, vars)
		if err != nil {
			return 0, fmt.Errorf("expand arg %q: %w", raw, err)
		}
		args = append(args, s)
	}

	cmd := exec.Command(manifest.Command, args...)
	cmd.Dir = cwd
	cmd.Env = buildChatEnv(manifest, topicID, topicStateDir, projectDir, cm.baseURL)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return 0, err
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return 0, err
	}
	// stderr to file in topic state dir.
	stderrFile, err := os.Create(filepath.Join(topicStateDir, "stderr.log"))
	if err != nil {
		return 0, err
	}
	cmd.Stderr = stderrFile

	if err := cmd.Start(); err != nil {
		_ = stderrFile.Close()
		return 0, err
	}

	// Build history payload for chat.start.
	type historyEntry struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	histEntries := make([]historyEntry, 0, len(history)+1)
	if instructions := loadDefaultInstructions(cm.dirs, "chat", manifest.Command); instructions != "" {
		histEntries = append(histEntries, historyEntry{Role: "system", Content: instructions})
	}
	for _, m := range history {
		histEntries = append(histEntries, historyEntry{Role: m.Role, Content: m.Content})
	}

	// Send chat.start — keep stdin OPEN.
	payload := map[string]any{
		"jsonrpc": "2.0",
		"method":  "chat.start",
		"params": map[string]any{
			"topic_id":    topicID,
			"project_dir": projectDir,
			"history":     histEntries,
		},
	}
	if err := json.NewEncoder(stdin).Encode(payload); err != nil {
		_ = cmd.Process.Kill()
		_ = stderrFile.Close()
		return 0, fmt.Errorf("write chat.start: %w", err)
	}

	readyCh := make(chan struct{}, 1)
	failCh := make(chan error, 1)

	// Stdout reader goroutine. Doubles as the ready-signal detector
	// during startup, then continues reading chat.reply messages.
	outCtx := context.Background()
	go func() {
		defer stderrFile.Close()
		scanner := bufio.NewScanner(stdoutPipe)
		scanner.Buffer(make([]byte, 1<<16), 1<<22)
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}
			slog.Debug("chat stdout", "topic_id", topicID, "line", string(line))
			var msg struct {
				Method string          `json:"method"`
				Params json.RawMessage `json:"params"`
			}
			if err := json.Unmarshal(line, &msg); err != nil {
				continue
			}
			switch msg.Method {
			case "worker.ready":
				select {
				case readyCh <- struct{}{}:
				default:
				}
			case "chat.reply":
				var p struct {
					InReplyTo string `json:"in_reply_to"`
					Content   string `json:"content"`
				}
				if err := json.Unmarshal(msg.Params, &p); err != nil {
					slog.Warn("chat.reply unmarshal", "topic_id", topicID, "err", err)
					continue
				}
				msgID, err := uuid.NewV7()
				if err != nil {
					slog.Warn("uuid for chat.reply", "topic_id", topicID, "err", err)
					continue
				}
				var replyTo *string
				if p.InReplyTo != "" {
					replyTo = &p.InReplyTo
				}
				if _, err := cm.store.CreateMessage(outCtx, msgID.String(), store.MessageCreate{
					TopicID:   topicID,
					Role:      "assistant",
					Content:   p.Content,
					InReplyTo: replyTo,
				}); err != nil {
					slog.Warn("save chat.reply", "topic_id", topicID, "err", err)
				}
			}
		}
		// Process exited — clean up.
		slog.Info("chat process exited", "topic_id", topicID)
		_ = cm.store.UpdateTopicPID(outCtx, topicID, nil)
		cm.mu.Lock()
		delete(cm.sessions, topicID)
		cm.mu.Unlock()
	}()

	// Watcher: detect early exit before ready.
	go func() {
		err := cmd.Wait()
		if err != nil {
			failCh <- fmt.Errorf("process exited: %w", err)
		} else {
			failCh <- nil
		}
	}()

	// Wait for ready signal, up to 10s.
	maxReady := 10 * time.Second
	select {
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		return 0, ctx.Err()
	case <-readyCh:
		pid = cmd.Process.Pid
		session := &ChatSession{
			TopicID: topicID,
			PID:     pid,
			Cmd:     cmd,
			Stdin:   stdin,
		}
		cm.mu.Lock()
		cm.sessions[topicID] = session
		cm.mu.Unlock()
		return pid, nil
	case err := <-failCh:
		if err == nil {
			return 0, errors.New("chat agent exited before becoming ready")
		}
		return 0, err
	case <-time.After(maxReady):
		_ = cmd.Process.Kill()
		return 0, fmt.Errorf("chat agent did not become ready within %s", maxReady)
	}
}

// Send writes a chat.message JSON-RPC line to the agent's stdin.
func (cm *ChatManager) Send(topicID, messageID, content string) error {
	cm.mu.RLock()
	sess, ok := cm.sessions[topicID]
	cm.mu.RUnlock()
	if !ok {
		return errors.New("no active session for topic")
	}
	payload := map[string]any{
		"jsonrpc": "2.0",
		"method":  "chat.message",
		"params": map[string]any{
			"message_id": messageID,
			"content":    content,
		},
	}
	sess.mu.Lock()
	defer sess.mu.Unlock()
	return json.NewEncoder(sess.Stdin).Encode(payload)
}

// Close sends SIGTERM to the chat agent and cleans up.
func (cm *ChatManager) Close(topicID string) error {
	cm.mu.Lock()
	sess, ok := cm.sessions[topicID]
	if ok {
		delete(cm.sessions, topicID)
	}
	cm.mu.Unlock()
	if !ok {
		return nil // already gone
	}
	if sess.Cmd.Process != nil {
		_ = syscall.Kill(sess.PID, syscall.SIGTERM)
	}
	_ = sess.Stdin.Close()
	return nil
}

// IsAlive checks if a chat session exists and the process responds to
// signal 0.
func (cm *ChatManager) IsAlive(topicID string) bool {
	cm.mu.RLock()
	sess, ok := cm.sessions[topicID]
	cm.mu.RUnlock()
	if !ok {
		return false
	}
	return syscall.Kill(sess.PID, 0) == nil
}

// RecoverOnStartup clears stale PIDs for any topics that were open
// when the server last shut down. Called once at boot.
func (cm *ChatManager) RecoverOnStartup(ctx context.Context) error {
	topics, err := cm.store.ListOpenTopics(ctx)
	if err != nil {
		return err
	}
	for _, t := range topics {
		if t.PID != nil {
			slog.Info("clearing stale chat PID", "topic_id", t.ID, "pid", *t.PID)
			_ = cm.store.UpdateTopicPID(ctx, t.ID, nil)
		}
	}
	return nil
}

// LoadManifestByName finds a manifest by name from the configured dirs.
func (cm *ChatManager) LoadManifestByName(name string) (*Manifest, error) {
	all, err := ScanDirs(cm.dirs.list())
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

// buildChatEnv composes the env slice for a chat agent process.
func buildChatEnv(manifest *Manifest, topicID, topicStateDir, projectDir, baseURL string) []string {
	env := os.Environ()
	env = append(env,
		"AC_RUN_ID="+topicID,
		"AC_TOPIC_DIR="+topicStateDir,
		"AC_BRAIN_DIR="+projectDir,
		"AC_CALLBACK_URL="+baseURL,
		"AC_API_URL="+baseURL+"/api",
	)
	vars := map[string]string{
		"topic.dir": topicStateDir,
		"brain.dir": projectDir,
	}
	for k, v := range manifest.Env {
		expanded, err := ExpandPlaceholders(v, vars)
		if err != nil {
			expanded = v
		}
		env = append(env, k+"="+expanded)
	}
	return env
}

func loadAgentSkills(manifest *Manifest) string {
	skillsDir := filepath.Join(filepath.Dir(manifest.Path()), "skills")
	var out []string
	if err := filepath.WalkDir(skillsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			slog.Warn("read chat skill", "path", path, "err", err)
			return nil
		}
		out = append(out, string(data))
		return nil
	}); err != nil && !errors.Is(err, fs.ErrNotExist) {
		slog.Warn("walk chat skills", "dir", skillsDir, "err", err)
	}
	if len(out) == 0 {
		return ""
	}
	return "Chat agent skills loaded from " + skillsDir + ":\n\n" + strings.Join(out, "\n\n---\n\n")
}
