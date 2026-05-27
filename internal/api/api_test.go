package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/Rachelint/agent-community/internal/plugin"
	"github.com/Rachelint/agent-community/internal/store"
)

func newTestAPI(t *testing.T) (*gin.Engine, *store.Store, *plugin.Manager) {
	t.Helper()
	repoAgentsDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoAgentsDir, "shared-skills", "issue-plan"), 0o755); err != nil {
		t.Fatalf("mkdir shared skills: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoAgentsDir, "shared-skills", "issue-plan", "SKILL.md"), []byte("skill"), 0o644); err != nil {
		t.Fatalf("write shared skill: %v", err)
	}
	return newTestAPIWithRepoAgents(t, repoAgentsDir)
}

func newTestAPIWithRepoAgents(t *testing.T, repoAgentsDir string) (*gin.Engine, *store.Store, *plugin.Manager) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	st, err := store.Open(context.Background(), filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	pm, err := plugin.NewManager(plugin.DataDirs{RepoAgentsDir: repoAgentsDir, UserAgentsDir: t.TempDir()}, "http://127.0.0.1:8080")
	if err != nil {
		t.Fatalf("new plugin manager: %v", err)
	}
	r := gin.New()
	Register(r, Deps{Store: st, Plugin: pm, WorkspacesDir: t.TempDir()})
	return r, st, pm
}

func requestJSON(t *testing.T, r http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestCreateProjectValidatesRepoLocal(t *testing.T) {
	r, _, _ := newTestAPI(t)

	w := requestJSON(t, r, http.MethodPost, "/api/projects", map[string]string{
		"name":       "bad",
		"repo_local": "relative/path",
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("bad repo status = %d, want 400: %s", w.Code, w.Body.String())
	}

	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	w = requestJSON(t, r, http.MethodPost, "/api/projects", map[string]string{
		"name":           "good",
		"repo_local":     repo,
		"default_branch": "main",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("good repo status = %d, want 201: %s", w.Code, w.Body.String())
	}
}

func TestCreateProjectInitializesRepoContext(t *testing.T) {
	repoAgentsDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoAgentsDir, "shared-skills", "issue-plan"), 0o755); err != nil {
		t.Fatalf("mkdir shared skills: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoAgentsDir, "shared-skills", "issue-plan", "SKILL.md"), []byte("skill"), 0o644); err != nil {
		t.Fatalf("write shared skill: %v", err)
	}
	r, _, _ := newTestAPIWithRepoAgents(t, repoAgentsDir)

	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".git", "info"), 0o755); err != nil {
		t.Fatalf("mkdir .git/info: %v", err)
	}
	w := requestJSON(t, r, http.MethodPost, "/api/projects", map[string]string{
		"name":           "good",
		"repo_local":     repo,
		"default_branch": "main",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create project status = %d, want 201: %s", w.Code, w.Body.String())
	}
	for _, rel := range []string{filepath.Join(".codebuddy", "skills"), filepath.Join(".claude", "skills")} {
		info, err := os.Lstat(filepath.Join(repo, rel))
		if err != nil {
			t.Fatalf("lstat %s: %v", rel, err)
		}
		if info.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("%s is not a symlink", rel)
		}
	}
	excludeData, err := os.ReadFile(filepath.Join(repo, ".git", "info", "exclude"))
	if err != nil {
		t.Fatalf("read exclude: %v", err)
	}
	if !strings.Contains(string(excludeData), ".codebuddy/") || !strings.Contains(string(excludeData), ".claude/") {
		t.Fatalf("exclude content = %q", string(excludeData))
	}
}

func requestPluginJSON(t *testing.T, r http.Handler, token, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func decodeJSON[T any](t *testing.T, body *bytes.Buffer) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(body.Bytes(), &out); err != nil {
		t.Fatalf("decode body: %v; body=%s", err, body.String())
	}
	return out
}

func TestPluginProjectsList(t *testing.T) {
	r, st, pm := newTestAPI(t)
	ctx := context.Background()
	if _, err := st.CreateProject(ctx, "project-1", store.ProjectCreate{Name: "one", RepoLocal: t.TempDir(), DefaultBranch: "main"}); err != nil {
		t.Fatalf("CreateProject p1: %v", err)
	}
	if _, err := st.CreateProject(ctx, "project-2", store.ProjectCreate{Name: "two", RepoLocal: t.TempDir(), DefaultBranch: "main"}); err != nil {
		t.Fatalf("CreateProject p2: %v", err)
	}

	w := requestPluginJSON(t, r, pm.Token(), http.MethodGet, "/plugin/projects", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list projects status = %d, want 200: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Items []map[string]any `json:"items"`
	}
	resp = decodeJSON[struct {
		Items []map[string]any `json:"items"`
	}](t, w.Body)
	if len(resp.Items) != 2 {
		t.Fatalf("project count = %d, want 2", len(resp.Items))
	}
	for _, item := range resp.Items {
		if len(item) != 2 {
			t.Fatalf("project item should only contain id/name: %+v", item)
		}
		if item["id"] == "" || item["name"] == "" {
			t.Fatalf("project item missing id/name: %+v", item)
		}
	}
}

func TestPluginCreateIssue(t *testing.T) {
	r, st, pm := newTestAPI(t)
	ctx := context.Background()
	project, err := st.CreateProject(ctx, "project-1", store.ProjectCreate{Name: "one", RepoLocal: t.TempDir(), DefaultBranch: "main"})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	w := requestPluginJSON(t, r, pm.Token(), http.MethodPost, "/plugin/projects/"+project.ID+"/issues", map[string]string{
		"title": "New issue",
		"body":  "Long body should not echo back.",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create issue status = %d, want 201: %s", w.Code, w.Body.String())
	}
	resp := decodeJSON[struct {
		OK          bool   `json:"ok"`
		IssueID     string `json:"issue_id"`
		IssueNumber int    `json:"issue_number"`
		Title       string `json:"title"`
		Body        string `json:"body"`
	}](t, w.Body)
	if !resp.OK || resp.IssueID == "" || resp.IssueNumber != 1 {
		t.Fatalf("unexpected create issue response: %+v", resp)
	}
	if resp.Title != "" || resp.Body != "" {
		t.Fatalf("create issue response should not echo title/body: %+v", resp)
	}
	iss, err := st.GetIssue(ctx, resp.IssueID)
	if err != nil {
		t.Fatalf("GetIssue: %v", err)
	}
	if iss.Title != "New issue" || iss.Body != "Long body should not echo back." {
		t.Fatalf("stored issue = %+v", iss)
	}
}

func TestPluginCreateIssueErrors(t *testing.T) {
	r, st, pm := newTestAPI(t)
	ctx := context.Background()
	project, err := st.CreateProject(ctx, "project-1", store.ProjectCreate{Name: "one", RepoLocal: t.TempDir(), DefaultBranch: "main"})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	t.Run("missing title", func(t *testing.T) {
		w := requestPluginJSON(t, r, pm.Token(), http.MethodPost, "/plugin/projects/"+project.ID+"/issues", map[string]string{"body": "x"})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400: %s", w.Code, w.Body.String())
		}
		resp := decodeJSON[struct {
			OK    bool `json:"ok"`
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
				Hint    string `json:"hint"`
			} `json:"error"`
		}](t, w.Body)
		if resp.OK || resp.Error.Code != "invalid_request" || resp.Error.Message != "title is required" || resp.Error.Hint == "" {
			t.Fatalf("unexpected error response: %+v", resp)
		}
	})

	t.Run("project not found", func(t *testing.T) {
		w := requestPluginJSON(t, r, pm.Token(), http.MethodPost, "/plugin/projects/missing/issues", map[string]string{"title": "x"})
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404: %s", w.Code, w.Body.String())
		}
		resp := decodeJSON[struct {
			OK    bool `json:"ok"`
			Error struct {
				Code string `json:"code"`
				Hint string `json:"hint"`
			} `json:"error"`
		}](t, w.Body)
		if resp.OK || resp.Error.Code != "project_not_found" || resp.Error.Hint == "" {
			t.Fatalf("unexpected error response: %+v", resp)
		}
	})

	t.Run("unauthorized", func(t *testing.T) {
		w := requestPluginJSON(t, r, "wrong-token", http.MethodGet, "/plugin/projects", nil)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401: %s", w.Code, w.Body.String())
		}
		resp := decodeJSON[struct {
			OK    bool `json:"ok"`
			Error struct {
				Code string `json:"code"`
				Hint string `json:"hint"`
			} `json:"error"`
		}](t, w.Body)
		if resp.OK || resp.Error.Code != "unauthorized" || resp.Error.Hint == "" {
			t.Fatalf("unexpected auth response: %+v", resp)
		}
	})
}

func TestPluginRunCallbacks(t *testing.T) {
	r, st, pm := newTestAPI(t)
	ctx := context.Background()
	project, err := st.CreateProject(ctx, "project-1", store.ProjectCreate{Name: "one", RepoLocal: t.TempDir(), DefaultBranch: "main"})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	issue, err := st.CreateIssue(ctx, "issue-1", project.ID, store.IssueCreate{Title: "issue"})
	if err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}
	runDir := t.TempDir()
	run, err := st.CreateRun(ctx, "run-1", store.RunCreate{IssueID: issue.ID, ProjectID: project.ID, Plugin: "worker-echo", WorkspaceDir: runDir})
	if err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	if err := st.MarkRunSpawned(ctx, run.ID, 12345); err != nil {
		t.Fatalf("MarkRunSpawned: %v", err)
	}

	w := requestPluginJSON(t, r, pm.Token(), http.MethodPost, "/plugin/runs/"+run.ID+"/ready", map[string]string{})
	if w.Code != http.StatusNoContent {
		t.Fatalf("ready status = %d, want 204: %s", w.Code, w.Body.String())
	}
	got, err := st.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRun after ready: %v", err)
	}
	if got.Status != store.RunRunning {
		t.Fatalf("run after ready = %+v", got)
	}

	w = requestPluginJSON(t, r, pm.Token(), http.MethodPost, "/plugin/runs/"+run.ID+"/log", map[string]string{
		"stream": "events",
		"data":   "{\"event\":\"test\"}\n",
	})
	if w.Code != http.StatusNoContent {
		t.Fatalf("log status = %d, want 204: %s", w.Code, w.Body.String())
	}
	logData, err := os.ReadFile(filepath.Join(runDir, "events.ndjson"))
	if err != nil {
		t.Fatalf("read events log: %v", err)
	}
	if string(logData) != "{\"event\":\"test\"}\n" {
		t.Fatalf("events log = %q", string(logData))
	}

	w = requestPluginJSON(t, r, pm.Token(), http.MethodPost, "/plugin/runs/"+run.ID+"/complete", map[string]any{
		"status":    "needs_review",
		"exit_code": 0,
		"summary":   "done",
		"mr_url":    "https://example.invalid/mr/1",
	})
	if w.Code != http.StatusNoContent {
		t.Fatalf("complete status = %d, want 204: %s", w.Code, w.Body.String())
	}
	got, err = st.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if got.Status != store.RunNeedsReview || got.Summary != "done" {
		t.Fatalf("run after complete = %+v", got)
	}
	count, err := st.CountUnreadNotifications(ctx, project.ID)
	if err != nil {
		t.Fatalf("CountUnreadNotifications: %v", err)
	}
	if count != 1 {
		t.Fatalf("notification count = %d, want 1", count)
	}

	w = requestPluginJSON(t, r, pm.Token(), http.MethodPost, "/plugin/runs/"+run.ID+"/fail", map[string]any{"summary": "late"})
	if w.Code != http.StatusConflict {
		t.Fatalf("late fail status = %d, want 409", w.Code)
	}
}

func TestCreateIssueValidatesParentAndLabels(t *testing.T) {
	r, st, _ := newTestAPI(t)
	ctx := context.Background()
	p1, err := st.CreateProject(ctx, "project-1", store.ProjectCreate{Name: "one", RepoLocal: t.TempDir(), DefaultBranch: "main"})
	if err != nil {
		t.Fatalf("CreateProject p1: %v", err)
	}
	p2, err := st.CreateProject(ctx, "project-2", store.ProjectCreate{Name: "two", RepoLocal: t.TempDir(), DefaultBranch: "main"})
	if err != nil {
		t.Fatalf("CreateProject p2: %v", err)
	}
	labelOther, err := st.CreateLabel(ctx, "label-other", p2.ID, store.LabelCreate{Name: "foreign", Color: "cccccc"})
	if err != nil {
		t.Fatalf("CreateLabel: %v", err)
	}
	parent, err := st.CreateIssue(ctx, "issue-parent", p1.ID, store.IssueCreate{Title: "parent"})
	if err != nil {
		t.Fatalf("CreateIssue parent: %v", err)
	}
	child, err := st.CreateIssue(ctx, "issue-child", p1.ID, store.IssueCreate{Title: "child", ParentID: &parent.ID})
	if err != nil {
		t.Fatalf("CreateIssue child: %v", err)
	}

	w := requestJSON(t, r, http.MethodPost, "/api/projects/"+p1.ID+"/issues", map[string]any{
		"title":     "grandchild",
		"parent_id": child.ID,
	})
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "single-level") {
		t.Fatalf("grandchild status/body = %d/%s, want single-level 400", w.Code, w.Body.String())
	}

	w = requestJSON(t, r, http.MethodPost, "/api/projects/"+p1.ID+"/issues", map[string]any{
		"title":  "wrong label",
		"labels": []string{labelOther.ID},
	})
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "different project") {
		t.Fatalf("foreign label status/body = %d/%s, want different-project 400", w.Code, w.Body.String())
	}
}
