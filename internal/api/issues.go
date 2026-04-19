package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Rachelint/agent-community/internal/store"
)

func registerIssues(g *gin.RouterGroup, st *store.Store) {
	g.GET("/projects/:pid/issues", func(c *gin.Context) {
		f := store.IssueFilter{
			Status:   c.Query("status"),
			Assignee: c.Query("assignee"),
			LabelID:  c.Query("label"),
			ParentID: c.Query("parent"),
			Q:        c.Query("q"),
		}
		is, err := st.ListIssues(c.Request.Context(), c.Param("pid"), f)
		if err != nil {
			internalError(c, err)
			return
		}
		if is == nil {
			is = []store.Issue{}
		}
		c.JSON(http.StatusOK, is)
	})

	g.POST("/projects/:pid/issues", func(c *gin.Context) {
		var req struct {
			Title    string   `json:"title" binding:"required"`
			Body     string   `json:"body"`
			ParentID string   `json:"parent_id"`
			Assignee string   `json:"assignee"`
			Labels   []string `json:"labels"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			badRequest(c, err.Error())
			return
		}
		pid := c.Param("pid")
		if _, err := st.GetProject(c.Request.Context(), pid); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
				return
			}
			internalError(c, err)
			return
		}
		if req.ParentID != "" {
			if err := validateParent(c, st, pid, req.ParentID, ""); err != nil {
				return
			}
		}
		if err := validateLabelsBelongTo(c, st, pid, req.Labels); err != nil {
			return
		}

		id, err := uuid.NewV7()
		if err != nil {
			internalError(c, err)
			return
		}
		in := store.IssueCreate{
			Title:    req.Title,
			Body:     req.Body,
			LabelIDs: req.Labels,
		}
		if req.ParentID != "" {
			p := req.ParentID
			in.ParentID = &p
		}
		if req.Assignee != "" {
			a := req.Assignee
			in.Assignee = &a
		}
		iss, err := st.CreateIssue(c.Request.Context(), id.String(), pid, in)
		if err != nil {
			internalError(c, err)
			return
		}
		c.JSON(http.StatusCreated, iss)
	})

	g.GET("/issues/:id", func(c *gin.Context) {
		iss, err := st.GetIssue(c.Request.Context(), c.Param("id"))
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if err != nil {
			internalError(c, err)
			return
		}
		c.JSON(http.StatusOK, iss)
	})

	g.PATCH("/issues/:id", func(c *gin.Context) {
		var req struct {
			Title    *string   `json:"title"`
			Body     *string   `json:"body"`
			Status   *string   `json:"status"`
			Assignee *string   `json:"assignee"`
			ParentID *string   `json:"parent_id"`
			Labels   *[]string `json:"labels"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			badRequest(c, err.Error())
			return
		}
		if req.Status != nil && *req.Status != "open" && *req.Status != "closed" {
			badRequest(c, "status must be open or closed")
			return
		}
		existing, err := st.GetIssue(c.Request.Context(), c.Param("id"))
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if err != nil {
			internalError(c, err)
			return
		}
		if req.ParentID != nil && *req.ParentID != "" {
			if err := validateParent(c, st, existing.ProjectID, *req.ParentID, existing.ID); err != nil {
				return
			}
		}
		if req.Labels != nil {
			if err := validateLabelsBelongTo(c, st, existing.ProjectID, *req.Labels); err != nil {
				return
			}
		}

		iss, err := st.UpdateIssue(c.Request.Context(), c.Param("id"), store.IssuePatch{
			Title:    req.Title,
			Body:     req.Body,
			Status:   req.Status,
			Assignee: req.Assignee,
			ParentID: req.ParentID,
			Labels:   req.Labels,
		})
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if err != nil {
			internalError(c, err)
			return
		}
		c.JSON(http.StatusOK, iss)
	})

	g.DELETE("/issues/:id", func(c *gin.Context) {
		err := st.SoftDeleteIssue(c.Request.Context(), c.Param("id"))
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

	// Comments
	g.GET("/issues/:id/comments", func(c *gin.Context) {
		if _, err := st.GetIssue(c.Request.Context(), c.Param("id")); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			internalError(c, err)
			return
		}
		cs, err := st.ListIssueComments(c.Request.Context(), c.Param("id"))
		if err != nil {
			internalError(c, err)
			return
		}
		c.JSON(http.StatusOK, cs)
	})

	g.POST("/issues/:id/comments", func(c *gin.Context) {
		var req struct {
			Author string `json:"author"`
			Body   string `json:"body" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			badRequest(c, err.Error())
			return
		}
		if req.Author == "" {
			req.Author = "user"
		}
		if _, err := st.GetIssue(c.Request.Context(), c.Param("id")); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
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
		cm, err := st.CreateIssueComment(c.Request.Context(), id.String(), c.Param("id"), req.Author, req.Body)
		if err != nil {
			internalError(c, err)
			return
		}
		c.JSON(http.StatusCreated, cm)
	})
}

// validateParent ensures parent belongs to the same project, is not
// itself a child (single-level parent-child rule), and is not the issue
// being patched (selfID, empty for create). Writes response + returns
// error on failure.
func validateParent(c *gin.Context, st *store.Store, projectID, parentID, selfID string) error {
	if parentID == selfID {
		badRequest(c, "issue cannot be its own parent")
		return errValidation
	}
	parent, err := st.GetIssue(c.Request.Context(), parentID)
	if errors.Is(err, store.ErrNotFound) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "parent issue not found"})
		return errValidation
	}
	if err != nil {
		internalError(c, err)
		return err
	}
	if parent.ProjectID != projectID {
		badRequest(c, "parent must be in the same project")
		return errValidation
	}
	if parent.ParentID != nil {
		badRequest(c, "parent must itself be a top-level issue (single-level hierarchy)")
		return errValidation
	}
	return nil
}

func validateLabelsBelongTo(c *gin.Context, st *store.Store, projectID string, labelIDs []string) error {
	for _, lid := range labelIDs {
		l, err := st.GetLabel(c.Request.Context(), lid)
		if errors.Is(err, store.ErrNotFound) {
			badRequest(c, "label "+lid+" not found")
			return errValidation
		}
		if err != nil {
			internalError(c, err)
			return err
		}
		if l.ProjectID != projectID {
			badRequest(c, "label "+lid+" belongs to a different project")
			return errValidation
		}
	}
	return nil
}

// errValidation is a sentinel used only to abort handler execution after
// we've already written an HTTP response.
var errValidation = errors.New("validation failed")
