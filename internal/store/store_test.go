package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	st, err := Open(context.Background(), filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestOpenAppliesMigrations(t *testing.T) {
	st := newTestStore(t)

	var version int
	if err := st.DB.QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil {
		t.Fatalf("read schema version: %v", err)
	}
	if version != 5 {
		t.Fatalf("schema version = %d, want 5", version)
	}
}

func TestStoreIssueRunAndNotificationWorkflow(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	project, err := st.CreateProject(ctx, "project-1", ProjectCreate{
		Name:          "Test Project",
		RepoLocal:     t.TempDir(),
		DefaultBranch: "main",
	})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	label, err := st.CreateLabel(ctx, "label-1", project.ID, LabelCreate{Name: "bug", Color: "d73a4a"})
	if err != nil {
		t.Fatalf("CreateLabel: %v", err)
	}
	issue, err := st.CreateIssue(ctx, "issue-1", project.ID, IssueCreate{
		Title:    "Fix bug",
		Body:     "details",
		LabelIDs: []string{label.ID},
	})
	if err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}
	if issue.Number != 1 || len(issue.Labels) != 1 {
		t.Fatalf("issue number/labels = %d/%d, want 1/1", issue.Number, len(issue.Labels))
	}

	child, err := st.CreateIssue(ctx, "issue-2", project.ID, IssueCreate{Title: "Child", ParentID: &issue.ID})
	if err != nil {
		t.Fatalf("CreateIssue child: %v", err)
	}
	if child.Number != 2 || child.ParentID == nil || *child.ParentID != issue.ID {
		t.Fatalf("child issue not linked to parent: %+v", child)
	}

	run, err := st.CreateRun(ctx, "run-1", RunCreate{
		IssueID:      issue.ID,
		ProjectID:    project.ID,
		Plugin:       "worker-echo",
		WorkspaceDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	if run.Status != RunQueued {
		t.Fatalf("run status = %s, want queued", run.Status)
	}
	if err := st.MarkRunRunning(ctx, run.ID, 12345); err != nil {
		t.Fatalf("MarkRunRunning: %v", err)
	}
	running, err := st.RunningRunForIssue(ctx, issue.ID)
	if err != nil {
		t.Fatalf("RunningRunForIssue: %v", err)
	}
	if running == nil || running.ID != run.ID {
		t.Fatalf("running run = %+v, want %s", running, run.ID)
	}

	exitCode := 0
	if err := st.FinishRun(ctx, run.ID, RunCompleted, "", "done", &exitCode); err != nil {
		t.Fatalf("FinishRun: %v", err)
	}
	if err := st.FinishRun(ctx, run.ID, RunCompleted, "", "again", &exitCode); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second FinishRun error = %v, want ErrNotFound", err)
	}
	running, err = st.RunningRunForIssue(ctx, issue.ID)
	if err != nil {
		t.Fatalf("RunningRunForIssue after finish: %v", err)
	}
	if running != nil {
		t.Fatalf("running run after finish = %+v, want nil", running)
	}

	orphan, err := st.CreateRun(ctx, "run-2", RunCreate{
		IssueID:      issue.ID,
		ProjectID:    project.ID,
		Plugin:       "worker-echo",
		WorkspaceDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("CreateRun orphan: %v", err)
	}
	if err := st.MarkRunRunning(ctx, orphan.ID, 54321); err != nil {
		t.Fatalf("MarkRunRunning orphan: %v", err)
	}
	if err := st.MarkRunOrphan(ctx, orphan.ID); err != nil {
		t.Fatalf("MarkRunOrphan: %v", err)
	}
	if err := st.MarkRunOrphan(ctx, orphan.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second MarkRunOrphan error = %v, want ErrNotFound", err)
	}

	orphanRun, err := st.GetRun(ctx, orphan.ID)
	if err != nil {
		t.Fatalf("GetRun orphan: %v", err)
	}
	if err := st.CreateRunNotification(ctx, *orphanRun, RunOrphan, "needs review", ""); err != nil {
		t.Fatalf("CreateRunNotification: %v", err)
	}
	count, err := st.CountUnreadNotifications(ctx, project.ID)
	if err != nil {
		t.Fatalf("CountUnreadNotifications: %v", err)
	}
	if count != 1 {
		t.Fatalf("notification count = %d, want 1", count)
	}
}
