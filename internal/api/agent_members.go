package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Rachelint/agent-community/internal/store"
)

func registerAgentMembers(g *gin.RouterGroup, st *store.Store) {
	g.GET("/agent_members", func(c *gin.Context) {
		ms, err := st.ListAgentMembers(c.Request.Context(), c.Query("kind"))
		if err != nil {
			internalError(c, err)
			return
		}
		if ms == nil {
			ms = []store.AgentMember{}
		}
		c.JSON(http.StatusOK, ms)
	})

	// reload will rescan agent.json manifests on disk and reconcile them
	// with the table. Implemented in phase 3; stubbed now so the client
	// can wire the UI without a follow-up server change.
	g.POST("/agent_members/reload", func(c *gin.Context) {
		c.JSON(http.StatusAccepted, gin.H{
			"status":  "not_implemented",
			"message": "manifest reload arrives in phase 3",
		})
	})
}
