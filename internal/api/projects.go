package api

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Rachelint/agent-community/internal/store"
)

func registerProjects(g *gin.RouterGroup, st *store.Store) {
	g.GET("/projects", func(c *gin.Context) {
		ps, err := st.ListProjects(c.Request.Context())
		if err != nil {
			internalError(c, err)
			return
		}
		if ps == nil {
			ps = []store.Project{}
		}
		c.JSON(http.StatusOK, ps)
	})

	g.POST("/projects", func(c *gin.Context) {
		var req struct {
			Name          string `json:"name" binding:"required"`
			RepoURL       string `json:"repo_url"`
			RepoLocal     string `json:"repo_local" binding:"required"`
			DefaultBranch string `json:"default_branch"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			badRequest(c, err.Error())
			return
		}
		// Validate repo_local exists and looks like a git checkout.
		if err := validateRepoLocal(req.RepoLocal); err != nil {
			badRequest(c, err.Error())
			return
		}
		id, err := uuid.NewV7()
		if err != nil {
			internalError(c, err)
			return
		}
		p, err := st.CreateProject(c.Request.Context(), id.String(), store.ProjectCreate{
			Name:          req.Name,
			RepoURL:       req.RepoURL,
			RepoLocal:     req.RepoLocal,
			DefaultBranch: req.DefaultBranch,
		})
		if err != nil {
			internalError(c, err)
			return
		}
		c.JSON(http.StatusCreated, p)
	})

	g.GET("/projects/:pid", func(c *gin.Context) {
		p, err := st.GetProject(c.Request.Context(), c.Param("pid"))
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if err != nil {
			internalError(c, err)
			return
		}
		c.JSON(http.StatusOK, p)
	})

	g.DELETE("/projects/:pid", func(c *gin.Context) {
		err := st.DeleteProject(c.Request.Context(), c.Param("pid"))
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if err != nil {
			internalError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})
}

// validateRepoLocal ensures the path exists, is a directory, and contains
// a .git entry (file or dir; git worktrees use a file). It deliberately
// does not run `git` — we only need a cheap sanity check here.
func validateRepoLocal(path string) error {
	if !filepath.IsAbs(path) {
		return errors.New("repo_local must be an absolute path")
	}
	fi, err := os.Stat(path)
	if err != nil {
		return errors.New("repo_local does not exist")
	}
	if !fi.IsDir() {
		return errors.New("repo_local is not a directory")
	}
	if _, err := os.Stat(filepath.Join(path, ".git")); err != nil {
		return errors.New("repo_local is not a git checkout (.git missing)")
	}
	return nil
}
