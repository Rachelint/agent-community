// Package api wires HTTP handlers onto a Gin engine.
//
// Phase 0 only exposes /healthz. Later phases add REST routes under /api
// and a WebSocket endpoint at /ws.
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Register attaches all routes to the given router.
func Register(r *gin.Engine) {
	r.GET("/healthz", healthz)

	// Placeholder groups; populated in later phases.
	r.Group("/api")
	r.Group("/plugin")
}

func healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
