// Package reconcile implements startup recovery for worker runs whose
// callback may have been missed while the server was down.
package reconcile

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/Rachelint/agent-community/internal/plugin"
	"github.com/Rachelint/agent-community/internal/store"
)

// doneFile is the schema of done.json written by the worker's outer
// script on exit.
type doneFile struct {
	Status   string `json:"status"` // "completed" | "needs_review" | "failed"
	ExitCode int    `json:"exit_code"`
	Summary  string `json:"summary,omitempty"`
	MRURL    string `json:"mr_url,omitempty"`
}

// Reconciler performs a single startup recovery pass.
type Reconciler struct {
	store  *store.Store
	plugin *plugin.Manager
}

// New creates a startup reconciler.
func New(s *store.Store, pm *plugin.Manager) *Reconciler {
	return &Reconciler{store: s, plugin: pm}
}

// RecoverAll checks active runs once during server startup.
func (r *Reconciler) RecoverAll() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	runs, err := r.store.ListActiveRuns(ctx)
	if err != nil {
		slog.Error("reconciler: list active runs", "err", err)
		return
	}

	for _, run := range runs {
		r.reconcileRun(ctx, run)
	}
}

func (r *Reconciler) reconcileRun(ctx context.Context, run store.WorkerRun) {
	donePath := filepath.Join(run.WorkspaceDir, "done.json")

	data, err := os.ReadFile(donePath)
	if err == nil {
		// done.json exists — parse and finish the run.
		var df doneFile
		if err := json.Unmarshal(data, &df); err != nil {
			slog.Warn("reconciler: invalid done.json",
				"run_id", run.ID, "err", err)
			// Treat as failed with the raw content as summary.
			df = doneFile{Status: "failed", ExitCode: 1, Summary: "invalid done.json: " + err.Error()}
		}
		r.finishRun(ctx, run, df)
		return
	}

	if !os.IsNotExist(err) {
		slog.Error("reconciler: read done.json", "run_id", run.ID, "err", err)
		return
	}

	// No done.json — check if the process is still alive.
	pid := 0
	if run.PID != nil {
		pid = *run.PID
	}
	alive, _ := r.plugin.Probe(run.ID, pid)
	if !alive {
		// Process died without writing done.json. Keep the result explicit:
		// mark it orphan so the user can inspect logs before deciding.
		slog.Info("reconciler: process dead without done.json",
			"run_id", run.ID, "pid", pid)
		r.markOrphan(ctx, run, "process exited without writing done.json")
	}
	// Otherwise still active — skip and wait for worker callbacks.
}

func (r *Reconciler) finishRun(ctx context.Context, run store.WorkerRun, df doneFile) {
	status := mapStatus(df.Status)
	exitCode := df.ExitCode
	if err := r.store.FinishRun(ctx, run.ID, status, df.MRURL, df.Summary, &exitCode); err != nil {
		slog.Error("reconciler: finish run", "run_id", run.ID, "err", err)
		return
	}
	r.plugin.Forget(run.ID)
	if err := r.store.CreateRunNotification(ctx, run, status, df.Summary, df.MRURL); err != nil {
		slog.Error("reconciler: create notification", "run_id", run.ID, "err", err)
		return
	}

	slog.Info("reconciler: run finished",
		"run_id", run.ID, "status", status, "exit_code", exitCode)
}

func (r *Reconciler) markOrphan(ctx context.Context, run store.WorkerRun, summary string) {
	if err := r.store.MarkRunOrphan(ctx, run.ID); err != nil {
		slog.Error("reconciler: mark orphan", "run_id", run.ID, "err", err)
		return
	}
	r.plugin.Forget(run.ID)
	if err := r.store.CreateRunNotification(ctx, run, store.RunOrphan, summary, ""); err != nil {
		slog.Error("reconciler: create orphan notification", "run_id", run.ID, "err", err)
		return
	}
	slog.Info("reconciler: run orphaned", "run_id", run.ID)
}

func mapStatus(s string) string {
	switch s {
	case "completed":
		return store.RunCompleted
	case "needs_review":
		return store.RunNeedsReview
	default:
		return store.RunFailed
	}
}
