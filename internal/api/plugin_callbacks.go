package api

import (
	"errors"
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
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "plugin auth not configured"})
			return
		}
		var got string
		if h := c.GetHeader("Authorization"); strings.HasPrefix(h, "Bearer ") {
			got = strings.TrimPrefix(h, "Bearer ")
		} else {
			got = c.Query("token")
		}
		if got != want {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Next()
	}
}

func registerPluginCallbacks(g *gin.RouterGroup, d Deps) {
	// Append a log chunk. Plugins should POST JSON:
	//   { "stream": "stdout"|"stderr"|"events", "content": "..." }
	// We append as-is to the on-disk file; the UI reads via /api/runs/:id/logs.
	g.POST("/runs/:id/log", func(c *gin.Context) {
		var req struct {
			Stream  string `json:"stream"`
			Content string `json:"content" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			badRequest(c, err.Error())
			return
		}
		r, err := d.Store.GetRun(c.Request.Context(), c.Param("id"))
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if err != nil {
			internalError(c, err)
			return
		}
		name := "stdout.log"
		switch req.Stream {
		case "", "stdout":
			name = "stdout.log"
		case "stderr":
			name = "stderr.log"
		case "events":
			name = "events.ndjson"
		default:
			badRequest(c, "stream must be stdout|stderr|events")
			return
		}
		dir := filepath.Join(r.WorkspaceDir, "logs")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			internalError(c, err)
			return
		}
		f, err := os.OpenFile(
			filepath.Join(dir, name),
			os.O_APPEND|os.O_CREATE|os.O_WRONLY,
			0o644,
		)
		if err != nil {
			internalError(c, err)
			return
		}
		defer f.Close()
		if _, err := f.WriteString(req.Content); err != nil {
			internalError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})

	// Plugin signals completion. status is one of 'completed' |
	// 'needs_review' | 'failed'. We bucket failed into RunFailed and
	// the others into their namesakes.
	g.POST("/runs/:id/complete", func(c *gin.Context) {
		var req struct {
			Status  string `json:"status"`
			MRURL   string `json:"mr_url"`
			Summary string `json:"summary"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			badRequest(c, err.Error())
			return
		}
		status := store.RunCompleted
		switch req.Status {
		case "", "completed":
			status = store.RunCompleted
		case "needs_review":
			status = store.RunNeedsReview
		case "failed":
			status = store.RunFailed
		default:
			badRequest(c, "status must be completed|needs_review|failed")
			return
		}
		r, err := d.Store.GetRun(c.Request.Context(), c.Param("id"))
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if err != nil {
			internalError(c, err)
			return
		}
		code := 0
		if err := d.Store.FinishRun(
			c.Request.Context(), r.ID, status, req.MRURL, req.Summary, &code,
		); err != nil {
			// Idempotency: a late duplicate callback on an already
			// terminal run is a no-op, not an error.
			if errors.Is(err, store.ErrNotFound) {
				c.JSON(http.StatusOK, gin.H{"status": "already terminal"})
				return
			}
			internalError(c, err)
			return
		}
		// Always leave a notification so the user notices.
		nid, _ := uuid.NewV7()
		title := r.Plugin + " · " + status
		body := req.Summary
		if req.MRURL != "" {
			if body != "" {
				body += "\n\n"
			}
			body += "MR: " + req.MRURL
		}
		_, _ = d.Store.CreateNotification(c.Request.Context(), nid.String(), store.NotificationCreate{
			ProjectID: r.ProjectID,
			Kind:      "worker_" + status,
			IssueID:   r.IssueID,
			RunID:     r.ID,
			Title:     title,
			Body:      body,
		})
		c.Status(http.StatusNoContent)
	})

	// Plugin reports an unrecoverable error. Semantically identical to
	// complete{status:"failed"} but kept as a separate endpoint so
	// scripts don't need to form the JSON body.
	g.POST("/runs/:id/fail", func(c *gin.Context) {
		var req struct {
			Reason string `json:"reason"`
		}
		_ = c.ShouldBindJSON(&req)
		r, err := d.Store.GetRun(c.Request.Context(), c.Param("id"))
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if err != nil {
			internalError(c, err)
			return
		}
		code := 1
		_ = d.Store.FinishRun(
			c.Request.Context(), r.ID, store.RunFailed, "", req.Reason, &code,
		)
		nid, _ := uuid.NewV7()
		_, _ = d.Store.CreateNotification(c.Request.Context(), nid.String(), store.NotificationCreate{
			ProjectID: r.ProjectID,
			Kind:      "worker_failed",
			IssueID:   r.IssueID,
			RunID:     r.ID,
			Title:     r.Plugin + " · failed",
			Body:      req.Reason,
		})
		c.Status(http.StatusNoContent)
	})

	// Kept around to verify the token is valid; useful during
	// local plugin development.
	g.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"pong": time.Now().UnixMilli()})
	})
}
