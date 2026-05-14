package api

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

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

func registerPluginRoutes(g *gin.RouterGroup, d Deps) {
	g.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"pong": time.Now().UnixMilli()})
	})

	g.POST("/runs/:id/log", func(c *gin.Context) {
		run, ok := loadRunningRun(c, d.Store)
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
		run, ok := loadRunningRun(c, d.Store)
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
		run, ok := loadRunningRun(c, d.Store)
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

func loadRunningRun(c *gin.Context, st *store.Store) (*store.WorkerRun, bool) {
	run, err := st.GetRun(c.Request.Context(), c.Param("id"))
	if errors.Is(err, store.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
		return nil, false
	}
	if err != nil {
		internalError(c, err)
		return nil, false
	}
	if run.Status != store.RunRunning {
		c.JSON(http.StatusConflict, gin.H{"error": "run is not running"})
		return nil, false
	}
	return run, true
}

func finishRunFromPlugin(c *gin.Context, d Deps, run store.WorkerRun, status, mrURL, summary string, exitCode int) {
	if err := d.Store.FinishRun(c.Request.Context(), run.ID, status, mrURL, summary, &exitCode); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusConflict, gin.H{"error": "run is not running"})
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
