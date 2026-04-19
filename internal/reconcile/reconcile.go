// Package reconcile implements a background loop that detects worker
// completion by polling for done.json files and dead processes.
package reconcile

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

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

// Reconciler polls for worker run completions.
type Reconciler struct {
	store    *store.Store
	plugin   *plugin.Manager
	interval time.Duration
	stop     chan struct{}
	done     chan struct{}
}

// New creates a Reconciler. interval controls how often it checks.
func New(s *store.Store, pm *plugin.Manager, interval time.Duration) *Reconciler {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	return &Reconciler{
		store:    s,
		plugin:   pm,
		interval: interval,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// Run starts the reconcile loop in the background. Non-blocking.
func (r *Reconciler) Run() {
	go r.loop()
}

// Stop signals the reconciler to shut down and waits for it to finish.
func (r *Reconciler) Stop() {
	close(r.stop)
	<-r.done
}

func (r *Reconciler) loop() {
	defer close(r.done)

	// Run once immediately on startup.
	r.reconcileOnce()

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-r.stop:
			return
		case <-ticker.C:
			r.reconcileOnce()
		}
	}
}

func (r *Reconciler) reconcileOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	runs, err := r.store.ListRunningRuns(ctx)
	if err != nil {
		slog.Error("reconciler: list running runs", "err", err)
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
		// Process died without writing done.json — mark as failed.
		slog.Info("reconciler: process dead without done.json",
			"run_id", run.ID, "pid", pid)
		df := doneFile{
			Status:   "failed",
			ExitCode: 1,
			Summary:  "process exited without writing done.json",
		}
		r.finishRun(ctx, run, df)
	}
	// Otherwise still running — skip.
}

func (r *Reconciler) finishRun(ctx context.Context, run store.WorkerRun, df doneFile) {
	status := mapStatus(df.Status)
	exitCode := df.ExitCode
	if err := r.store.FinishRun(ctx, run.ID, status, df.MRURL, df.Summary, &exitCode); err != nil {
		slog.Error("reconciler: finish run", "run_id", run.ID, "err", err)
		return
	}

	// Create a notification.
	nid, err := uuid.NewV7()
	if err != nil {
		return
	}
	title := run.Plugin + " · " + status
	body := df.Summary
	if df.MRURL != "" {
		if body != "" {
			body += "\n\n"
		}
		body += "MR: " + df.MRURL
	}
	_, _ = r.store.CreateNotification(ctx, nid.String(), store.NotificationCreate{
		ProjectID: run.ProjectID,
		Kind:      "worker_" + status,
		IssueID:   run.IssueID,
		RunID:     run.ID,
		Title:     title,
		Body:      body,
	})

	slog.Info("reconciler: run finished",
		"run_id", run.ID, "status", status, "exit_code", exitCode)
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
