package api

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Rachelint/agent-community/internal/plugin"
	"github.com/Rachelint/agent-community/internal/store"
)

// pluginAuth gates /plugin/* endpoints behind the shared token generated
// by the plugin manager. Token is passed via Authorization: Bearer <t>
// or ?token=<t>.
func pluginAuth(pm *plugin.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		want := pm.Token()
		if want == "" {
			pluginError(c, http.StatusServiceUnavailable, "plugin_auth_not_configured", "plugin auth is not configured", "This is a server-side configuration issue. Ask the platform operator to enable plugin auth.")
			c.Abort()
			return
		}
		var got string
		if h := c.GetHeader("Authorization"); strings.HasPrefix(h, "Bearer ") {
			got = strings.TrimPrefix(h, "Bearer ")
		} else {
			got = c.Query("token")
		}
		if got != want {
			pluginError(c, http.StatusUnauthorized, "unauthorized", "missing or invalid plugin token", "Send Authorization: Bearer <AC_PLUGIN_TOKEN> and retry.")
			c.Abort()
			return
		}
		c.Next()
	}
}

func registerPluginRoutes(g *gin.RouterGroup, d Deps) {
	g.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"pong": time.Now().UnixMilli()})
	})

	g.GET("/projects", func(c *gin.Context) {
		projects, err := d.Store.ListProjects(c.Request.Context())
		if err != nil {
			slog.Error("plugin list projects", "path", c.FullPath(), "err", err)
			pluginError(c, http.StatusInternalServerError, "internal_error", "failed to list projects", "Retry once. If it still fails, check server logs or surface the error to the user.")
			return
		}
		items := make([]gin.H, 0, len(projects))
		for _, p := range projects {
			items = append(items, gin.H{"id": p.ID, "name": p.Name})
		}
		c.JSON(http.StatusOK, gin.H{"items": items})
	})

	g.POST("/projects/:pid/issues", func(c *gin.Context) {
		var req struct {
			Title string `json:"title"`
			Body  string `json:"body"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			pluginError(c, http.StatusBadRequest, "invalid_request", "invalid JSON body", "Send a JSON object with a non-empty title and optional body, then retry.")
			return
		}
		req.Title = strings.TrimSpace(req.Title)
		if req.Title == "" {
			pluginError(c, http.StatusBadRequest, "invalid_request", "title is required", "Provide a non-empty issue title and retry.")
			return
		}

		pid := c.Param("pid")
		if _, err := d.Store.GetProject(c.Request.Context(), pid); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				pluginError(c, http.StatusNotFound, "project_not_found", "project not found", "Call GET /plugin/projects to refresh the project list, then retry with a valid project id.")
				return
			}
			slog.Error("plugin get project", "path", c.FullPath(), "project_id", pid, "err", err)
			pluginError(c, http.StatusInternalServerError, "internal_error", "failed to load project", "Retry once. If it still fails, check server logs or surface the error to the user.")
			return
		}

		issueID, err := uuid.NewV7()
		if err != nil {
			slog.Error("plugin issue id", "path", c.FullPath(), "project_id", pid, "err", err)
			pluginError(c, http.StatusInternalServerError, "internal_error", "failed to create issue", "Retry once. If it still fails, check server logs or surface the error to the user.")
			return
		}
		iss, err := d.Store.CreateIssue(c.Request.Context(), issueID.String(), pid, store.IssueCreate{
			Title: req.Title,
			Body:  req.Body,
		})
		if err != nil {
			slog.Error("plugin create issue", "path", c.FullPath(), "project_id", pid, "err", err)
			pluginError(c, http.StatusInternalServerError, "internal_error", "failed to create issue", "Retry once. If it still fails, check server logs or surface the error to the user.")
			return
		}
		c.JSON(http.StatusCreated, gin.H{"ok": true, "issue_id": iss.ID, "issue_number": iss.Number})
	})

	g.POST("/runs/:id/ready", func(c *gin.Context) {
		run, ok := loadActiveRun(c, d.Store)
		if !ok {
			return
		}
		pid := 0
		if run.PID != nil {
			pid = *run.PID
		}
		if err := d.Store.MarkRunRunning(c.Request.Context(), run.ID, pid); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				c.JSON(http.StatusConflict, gin.H{"error": "run is not active"})
				return
			}
			internalError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})

	g.POST("/runs/:id/log", func(c *gin.Context) {
		run, ok := loadActiveRun(c, d.Store)
		if !ok {
			return
		}
		var req struct {
			Stream string `json:"stream"`
			Data   string `json:"data"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			badRequest(c, err.Error())
			return
		}
		name, err := logFileName(req.Stream)
		if err != nil {
			badRequest(c, err.Error())
			return
		}
		path := filepath.Join(run.WorkspaceDir, name)
		if err := appendRunLog(path, req.Data); err != nil {
			internalError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})

	g.POST("/runs/:id/complete", func(c *gin.Context) {
		run, ok := loadActiveRun(c, d.Store)
		if !ok {
			return
		}
		var req struct {
			Status   string `json:"status"`
			ExitCode int    `json:"exit_code"`
			Summary  string `json:"summary"`
			MRURL    string `json:"mr_url"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			badRequest(c, err.Error())
			return
		}
		status := req.Status
		if status == "" {
			status = store.RunCompleted
		}
		if status != store.RunCompleted && status != store.RunNeedsReview {
			badRequest(c, "status must be completed or needs_review")
			return
		}
		finishRunFromPlugin(c, d, *run, status, req.MRURL, req.Summary, req.ExitCode)
	})

	g.POST("/runs/:id/fail", func(c *gin.Context) {
		run, ok := loadActiveRun(c, d.Store)
		if !ok {
			return
		}
		var req struct {
			ExitCode int    `json:"exit_code"`
			Summary  string `json:"summary"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			badRequest(c, err.Error())
			return
		}
		exitCode := req.ExitCode
		if exitCode == 0 {
			exitCode = 1
		}
		finishRunFromPlugin(c, d, *run, store.RunFailed, "", req.Summary, exitCode)
	})
}

func loadActiveRun(c *gin.Context, st *store.Store) (*store.WorkerRun, bool) {
	run, err := st.GetRun(c.Request.Context(), c.Param("id"))
	if errors.Is(err, store.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
		return nil, false
	}
	if err != nil {
		internalError(c, err)
		return nil, false
	}
	if run.Status != store.RunQueued && run.Status != store.RunRunning {
		c.JSON(http.StatusConflict, gin.H{"error": "run is not active"})
		return nil, false
	}
	return run, true
}

func finishRunFromPlugin(c *gin.Context, d Deps, run store.WorkerRun, status, mrURL, summary string, exitCode int) {
	if err := d.Store.FinishRun(c.Request.Context(), run.ID, status, mrURL, summary, &exitCode); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusConflict, gin.H{"error": "run is not active"})
			return
		}
		internalError(c, err)
		return
	}
	d.Plugin.Forget(run.ID)
	if err := d.Store.CreateRunNotification(c.Request.Context(), run, status, summary, mrURL); err != nil {
		internalError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func logFileName(stream string) (string, error) {
	switch stream {
	case "", "stdout":
		return "stdout.log", nil
	case "stderr":
		return "stderr.log", nil
	case "events":
		return "events.ndjson", nil
	default:
		return "", errors.New("stream must be stdout|stderr|events")
	}
}

func appendRunLog(path, data string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(data)
	return err
}

func pluginError(c *gin.Context, status int, code, message, hint string) {
	c.JSON(status, gin.H{
		"ok": false,
		"error": gin.H{
			"code":    code,
			"message": message,
			"hint":    hint,
		},
	})
}
