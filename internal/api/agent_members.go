package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Rachelint/agent-community/internal/store"
)

func registerAgentMembers(g *gin.RouterGroup, d Deps) {
	g.GET("/agent_members", func(c *gin.Context) {
		ms, err := d.Store.ListAgentMembers(c.Request.Context(), c.Query("kind"))
		if err != nil {
			internalError(c, err)
			return
		}
		if ms == nil {
			ms = []store.AgentMember{}
		}
		c.JSON(http.StatusOK, ms)
	})

	// Reload reconciles on-disk manifests with the agent_members table.
	// Policy:
	//   - Manifests present on disk → UPSERT (enabled preserved if
	//     already present, defaulted to true otherwise).
	//   - Registrations whose manifest_path disappeared from disk →
	//     row kept (historical runs may reference the name) but flagged
	//     enabled=false so it no longer appears in dispatch selectors.
	g.POST("/agent_members/reload", func(c *gin.Context) {
		manifests, err := d.Plugin.ScanManifests()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		seen := make(map[string]bool, len(manifests))
		for _, m := range manifests {
			seen[m.Name] = true
			if err := d.Store.UpsertAgentMember(
				c.Request.Context(), m.Name, m.Kind, m.Path(),
			); err != nil {
				internalError(c, err)
				return
			}
		}

		// Disable registrations whose manifest file is missing.
		existing, err := d.Store.ListAgentMembers(c.Request.Context(), "")
		if err != nil {
			internalError(c, err)
			return
		}
		var disabled []string
		for _, m := range existing {
			if !m.Enabled {
				continue
			}
			if !seen[m.Name] {
				if err := d.Store.DisableAgentMember(c.Request.Context(), m.Name); err != nil {
					internalError(c, err)
					return
				}
				disabled = append(disabled, m.Name)
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"loaded":   len(manifests),
			"disabled": disabled,
		})
	})
}
