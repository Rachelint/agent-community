// Package api wires HTTP handlers onto a Gin engine.
//
// Routes are organised into sub-files by resource (projects.go,
// agent_members.go, ...). Register is the single entry point called
// from cmd/server/main.go.
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Rachelint/agent-community/internal/plugin"
	"github.com/Rachelint/agent-community/internal/store"
)

// Deps bundles the handles every handler might need. Threading a
// single struct keeps handler signatures small as new cross-cutting
// dependencies (plugin manager, hub, ...) appear.
type Deps struct {
	Store         *store.Store
	Plugin        *plugin.Manager
	WorkspacesDir string
}

// Register attaches all routes to the given router.
func Register(r *gin.Engine, d Deps) {
	r.GET("/healthz", healthz)

	apiG := r.Group("/api")
	registerProjects(apiG, d.Store)
	registerAgentMembers(apiG, d)
	registerLabels(apiG, d.Store)
	registerIssues(apiG, d.Store)
	registerRuns(apiG, d)
	registerNotifications(apiG, d.Store)

	pluginG := r.Group("/plugin", pluginAuth(d.Plugin))
	registerPluginRoutes(pluginG)
}

func healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
