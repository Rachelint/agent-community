package reconcile

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rachelint/agent-community/internal/plugin"
	"github.com/Rachelint/agent-community/internal/store"
)

func newTestReconciler(t *testing.T) (*Reconciler, *store.Store, store.WorkerRun) {
	t.Helper()
	ctx := context.Background()
	st, err := store.Open(ctx, filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	pm, err := plugin.NewManager(plugin.DataDirs{RepoAgentsDir: t.TempDir(), UserAgentsDir: t.TempDir()}, "http://127.0.0.1:8080")
	if err != nil {
		t.Fatalf("new plugin manager: %v", err)
	}
	project, err := st.CreateProject(ctx, "project-1", store.ProjectCreate{Name: "one", RepoLocal: t.TempDir(), DefaultBranch: "main"})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	issue, err := st.CreateIssue(ctx, "issue-1", project.ID, store.IssueCreate{Title: "issue"})
	if err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}
	run, err := st.CreateRun(ctx, "run-1", store.RunCreate{
		IssueID:      issue.ID,
		ProjectID:    project.ID,
		Plugin:       "worker-echo",
		WorkspaceDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	if err := st.MarkRunRunning(ctx, run.ID, 0); err != nil {
		t.Fatalf("MarkRunRunning: %v", err)
	}
	running, err := st.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	return New(st, pm, time.Hour), st, *running
}

func TestReconcileDoneJSONFinishesRun(t *testing.T) {
	ctx := context.Background()
	rec, st, run := newTestReconciler(t)
	data := []byte(`{"status":"needs_review","exit_code":0,"summary":"ready","mr_url":"https://example.invalid/mr/1"}`)
	if err := os.WriteFile(filepath.Join(run.WorkspaceDir, "done.json"), data, 0o644); err != nil {
		t.Fatalf("write done.json: %v", err)
	}

	rec.reconcileRun(ctx, run)

	got, err := st.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if got.Status != store.RunNeedsReview || got.Summary != "ready" || got.MRURL == "" {
		t.Fatalf("run after reconcile = %+v", got)
	}
	count, err := st.CountUnreadNotifications(ctx, run.ProjectID)
	if err != nil {
		t.Fatalf("CountUnreadNotifications: %v", err)
	}
	if count != 1 {
		t.Fatalf("notification count = %d, want 1", count)
	}
}

func TestReconcileInvalidDoneJSONFailsRun(t *testing.T) {
	ctx := context.Background()
	rec, st, run := newTestReconciler(t)
	if err := os.WriteFile(filepath.Join(run.WorkspaceDir, "done.json"), []byte(`not json`), 0o644); err != nil {
		t.Fatalf("write done.json: %v", err)
	}

	rec.reconcileRun(ctx, run)

	got, err := st.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if got.Status != store.RunFailed || got.ExitCode == nil || *got.ExitCode != 1 {
		t.Fatalf("run after invalid done.json = %+v", got)
	}
}

func TestReconcileDeadProcessWithoutDoneMarksOrphan(t *testing.T) {
	ctx := context.Background()
	rec, st, run := newTestReconciler(t)

	rec.reconcileRun(ctx, run)

	got, err := st.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if got.Status != store.RunOrphan {
		t.Fatalf("run status = %s, want orphan", got.Status)
	}
	count, err := st.CountUnreadNotifications(ctx, run.ProjectID)
	if err != nil {
		t.Fatalf("CountUnreadNotifications: %v", err)
	}
	if count != 1 {
		t.Fatalf("notification count = %d, want 1", count)
	}
}
