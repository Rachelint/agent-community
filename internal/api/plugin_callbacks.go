package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Rachelint/agent-community/internal/plugin"
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

// registerPluginRoutes adds the minimal plugin endpoints (just ping for
// debugging/token-validation).
func registerPluginRoutes(g *gin.RouterGroup) {
	g.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"pong": time.Now().UnixMilli()})
	})
}
