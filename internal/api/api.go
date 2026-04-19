// Package api wires HTTP handlers onto a Gin engine.
//
// Routes are organised into sub-files by resource (projects.go,
// agent_members.go, ...). Register is the single entry point called from
// cmd/server/main.go.
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Rachelint/agent-community/internal/store"
)

// Register attaches all routes to the given router. The store pointer is
// threaded into handlers that need database access.
func Register(r *gin.Engine, st *store.Store) {
	r.GET("/healthz", healthz)

	apiG := r.Group("/api")
	registerProjects(apiG, st)
	registerAgentMembers(apiG, st)
	registerLabels(apiG, st)
	registerIssues(apiG, st)

	// Plugin callback endpoints (phase 3).
	r.Group("/plugin")
}

func healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
