package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Rachelint/agent-community/internal/store"
)

func registerLabels(g *gin.RouterGroup, st *store.Store) {
	g.GET("/projects/:pid/labels", func(c *gin.Context) {
		ls, err := st.ListLabels(c.Request.Context(), c.Param("pid"))
		if err != nil {
			internalError(c, err)
			return
		}
		if ls == nil {
			ls = []store.Label{}
		}
		c.JSON(http.StatusOK, ls)
	})

	g.POST("/projects/:pid/labels", func(c *gin.Context) {
		var req struct {
			Name        string `json:"name" binding:"required"`
			Color       string `json:"color" binding:"required"`
			Description string `json:"description"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			badRequest(c, err.Error())
			return
		}
		if _, err := st.GetProject(c.Request.Context(), c.Param("pid")); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
				return
			}
			internalError(c, err)
			return
		}
		id, err := uuid.NewV7()
		if err != nil {
			internalError(c, err)
			return
		}
		l, err := st.CreateLabel(c.Request.Context(), id.String(), c.Param("pid"), store.LabelCreate{
			Name: req.Name, Color: req.Color, Description: req.Description,
		})
		if errors.Is(err, store.ErrConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": "label name already exists in project"})
			return
		}
		if err != nil {
			internalError(c, err)
			return
		}
		c.JSON(http.StatusCreated, l)
	})

	g.PATCH("/labels/:id", func(c *gin.Context) {
		var req struct {
			Name        *string `json:"name"`
			Color       *string `json:"color"`
			Description *string `json:"description"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			badRequest(c, err.Error())
			return
		}
		l, err := st.UpdateLabel(c.Request.Context(), c.Param("id"), store.LabelPatch{
			Name: req.Name, Color: req.Color, Description: req.Description,
		})
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if errors.Is(err, store.ErrConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": "label name already exists in project"})
			return
		}
		if err != nil {
			internalError(c, err)
			return
		}
		c.JSON(http.StatusOK, l)
	})

	g.DELETE("/labels/:id", func(c *gin.Context) {
		if err := st.DeleteLabel(c.Request.Context(), c.Param("id")); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			internalError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})
}
