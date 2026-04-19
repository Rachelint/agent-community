package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Rachelint/agent-community/internal/plugin"
	"github.com/Rachelint/agent-community/internal/store"
)

func registerRuns(g *gin.RouterGroup, d Deps) {
	g.GET("/issues/:id/runs", func(c *gin.Context) {
		if _, err := d.Store.GetIssue(c.Request.Context(), c.Param("id")); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "issue not found"})
				return
			}
			internalError(c, err)
			return
		}
		rs, err := d.Store.ListRunsByIssue(c.Request.Context(), c.Param("id"))
		if err != nil {
			internalError(c, err)
			return
		}
		c.JSON(http.StatusOK, rs)
	})

	g.POST("/issues/:id/dispatch", func(c *gin.Context) {
		var req struct {
			Plugin string `json:"plugin" binding:"required"`
			Prompt string `json:"prompt"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			badRequest(c, err.Error())
			return
		}

		iss, err := d.Store.GetIssue(c.Request.Context(), c.Param("id"))
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "issue not found"})
			return
		}
		if err != nil {
			internalError(c, err)
			return
		}

		// Guard: at most one running run per issue. The user must
		// cancel or mark_orphan the existing one before starting again.
		if existing, err := d.Store.RunningRunForIssue(c.Request.Context(), iss.ID); err != nil {
			internalError(c, err)
			return
		} else if existing != nil {
			c.JSON(http.StatusConflict, gin.H{
				"error":      "another run is still running for this issue",
				"running_id": existing.ID,
			})
			return
		}

		// Require the plugin to exist, be enabled, and be a worker.
		mem, err := d.Store.GetAgentMember(c.Request.Context(), req.Plugin)
		if errors.Is(err, store.ErrNotFound) {
			badRequest(c, "unknown plugin: "+req.Plugin)
			return
		}
		if err != nil {
			internalError(c, err)
			return
		}
		if !mem.Enabled {
			badRequest(c, "plugin is disabled: "+req.Plugin)
			return
		}
		if mem.Kind != "worker" {
			badRequest(c, "plugin is not a worker: "+req.Plugin)
			return
		}

		proj, err := d.Store.GetProject(c.Request.Context(), iss.ProjectID)
		if err != nil {
			internalError(c, err)
			return
		}

		manifest, err := d.Plugin.LoadManifestByName(req.Plugin)
		if err != nil {
			internalError(c, fmt.Errorf("load manifest: %w", err))
			return
		}

		runID, err := uuid.NewV7()
		if err != nil {
			internalError(c, err)
			return
		}
		workspace := filepath.Join(d.WorkspacesDir, runID.String())
		if err := os.MkdirAll(workspace, 0o755); err != nil {
			internalError(c, err)
			return
		}
		prompt := req.Prompt
		if prompt == "" {
			prompt = iss.Title
			if iss.Body != "" {
				prompt += "\n\n" + iss.Body
			}
		}
		// Persist the prompt so a failed run still has the input
		// recorded.
		_ = os.WriteFile(filepath.Join(workspace, "prompt.md"), []byte(prompt), 0o644)

		run, err := d.Store.CreateRun(c.Request.Context(), runID.String(), store.RunCreate{
			IssueID:      iss.ID,
			ProjectID:    iss.ProjectID,
			Plugin:       req.Plugin,
			WorkspaceDir: workspace,
		})
		if err != nil {
			internalError(c, err)
			return
		}

		pid, err := d.Plugin.Spawn(c.Request.Context(), plugin.SpawnArgs{
			Manifest:     manifest,
			RunID:        run.ID,
			WorkspaceDir: workspace,
			BrainDir:     proj.RepoLocal,
			Prompt:       prompt,
		})
		if err != nil {
			// Flip straight to failed; no complete callback will arrive.
			code := -1
			_ = d.Store.MarkRunRunning(c.Request.Context(), run.ID, 0)
			_ = d.Store.FinishRun(c.Request.Context(), run.ID, store.RunFailed, "", err.Error(), &code)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":  "spawn failed",
				"detail": err.Error(),
				"run_id": run.ID,
			})
			return
		}
		if err := d.Store.MarkRunRunning(c.Request.Context(), run.ID, pid); err != nil {
			internalError(c, err)
			return
		}
		run.Status = store.RunRunning
		run.PID = &pid
		c.JSON(http.StatusAccepted, run)
	})

	g.GET("/runs/:id", func(c *gin.Context) {
		r, err := d.Store.GetRun(c.Request.Context(), c.Param("id"))
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if err != nil {
			internalError(c, err)
			return
		}
		c.JSON(http.StatusOK, r)
	})

	g.POST("/runs/:id/cancel", func(c *gin.Context) {
		r, err := d.Store.GetRun(c.Request.Context(), c.Param("id"))
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if err != nil {
			internalError(c, err)
			return
		}
		if r.Status != store.RunRunning {
			badRequest(c, "run is not running")
			return
		}
		pid := 0
		if r.PID != nil {
			pid = *r.PID
		}
		if err := d.Plugin.Cancel(r.ID, pid); err != nil {
			internalError(c, err)
			return
		}
		// Mark cancelled synchronously; the process exit callback (if
		// any) will find the row already terminal and be a no-op.
		code := -1
		if err := d.Store.FinishRun(
			c.Request.Context(), r.ID, store.RunCancelled, "", "cancelled by user", &code,
		); err != nil {
			internalError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})

	g.POST("/runs/:id/probe", func(c *gin.Context) {
		r, err := d.Store.GetRun(c.Request.Context(), c.Param("id"))
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if err != nil {
			internalError(c, err)
			return
		}
		pid := 0
		if r.PID != nil {
			pid = *r.PID
		}
		alive, checkedPID := d.Plugin.Probe(r.ID, pid)
		c.JSON(http.StatusOK, gin.H{
			"alive": alive,
			"pid":   checkedPID,
		})
	})

	g.POST("/runs/:id/mark_orphan", func(c *gin.Context) {
		err := d.Store.MarkRunOrphan(c.Request.Context(), c.Param("id"))
		if errors.Is(err, store.ErrNotFound) {
			badRequest(c, "run is not running or does not exist")
			return
		}
		if err != nil {
			internalError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})

	// Tail a log file. Params:
	//   stream: "stdout" (default) | "stderr" | "events"
	//   from:   byte offset (default 0)
	//   limit:  max bytes to read (default 64 KiB, capped at 1 MiB)
	g.GET("/runs/:id/logs", func(c *gin.Context) {
		r, err := d.Store.GetRun(c.Request.Context(), c.Param("id"))
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if err != nil {
			internalError(c, err)
			return
		}
		stream := c.DefaultQuery("stream", "stdout")
		var name string
		switch stream {
		case "stdout":
			name = "stdout.log"
		case "stderr":
			name = "stderr.log"
		case "events":
			name = "events.ndjson"
		default:
			badRequest(c, "stream must be stdout|stderr|events")
			return
		}
		from, _ := strconv.ParseInt(c.DefaultQuery("from", "0"), 10, 64)
		limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "65536"), 10, 64)
		if limit <= 0 {
			limit = 65536
		}
		if limit > 1<<20 {
			limit = 1 << 20
		}
		path := filepath.Join(r.WorkspaceDir, "logs", name)
		out, next, err := readLogRange(path, from, limit)
		if err != nil {
			internalError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"stream": stream,
			"from":   from,
			"next":   next,
			"chunk":  out,
		})
	})
}

// readLogRange returns a slice of the file between [from, from+limit).
// If the file doesn't exist yet (pre-first-write), returns empty chunk
// and next=from. Returns next = end of chunk in the file.
func readLogRange(path string, from, limit int64) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", from, nil
		}
		return "", 0, err
	}
	defer f.Close()
	if _, err := f.Seek(from, io.SeekStart); err != nil {
		return "", from, nil
	}
	buf := make([]byte, limit)
	n, err := f.Read(buf)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", 0, err
	}
	return string(buf[:n]), from + int64(n), nil
}
