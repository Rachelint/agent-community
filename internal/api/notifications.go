package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Rachelint/agent-community/internal/store"
)

func registerNotifications(g *gin.RouterGroup, s *store.Store) {
	// List notifications for a project.
	// GET /api/projects/:pid/notifications?unread=true
	g.GET("/projects/:pid/notifications", func(c *gin.Context) {
		pid := c.Param("pid")
		unread := c.Query("unread") == "true"
		ns, err := s.ListNotifications(c.Request.Context(), pid, unread)
		if err != nil {
			internalError(c, err)
			return
		}
		c.JSON(http.StatusOK, ns)
	})

	// Count unread notifications for a project.
	// GET /api/projects/:pid/notifications/count
	g.GET("/projects/:pid/notifications/count", func(c *gin.Context) {
		pid := c.Param("pid")
		count, err := s.CountUnreadNotifications(c.Request.Context(), pid)
		if err != nil {
			internalError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"count": count})
	})

	// Update a notification (mark read or archived).
	// PATCH /api/notifications/:id
	g.PATCH("/notifications/:id", func(c *gin.Context) {
		var req struct {
			Read     *bool `json:"read"`
			Archived *bool `json:"archived"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			badRequest(c, err.Error())
			return
		}
		id := c.Param("id")
		if req.Read != nil && *req.Read {
			if err := s.MarkNotificationRead(c.Request.Context(), id); err != nil {
				if errors.Is(err, store.ErrNotFound) {
					c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
					return
				}
				internalError(c, err)
				return
			}
		}
		if req.Archived != nil && *req.Archived {
			if err := s.ArchiveNotification(c.Request.Context(), id); err != nil {
				if errors.Is(err, store.ErrNotFound) {
					c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
					return
				}
				internalError(c, err)
				return
			}
		}
		c.Status(http.StatusNoContent)
	})
}
