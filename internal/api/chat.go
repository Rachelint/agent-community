package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Rachelint/agent-community/internal/store"
)

func registerChat(g *gin.RouterGroup, d Deps) {
	// List topics for a project.
	g.GET("/projects/:pid/topics", func(c *gin.Context) {
		topics, err := d.Store.ListTopicsByProject(c.Request.Context(), c.Param("pid"))
		if err != nil {
			internalError(c, err)
			return
		}
		c.JSON(http.StatusOK, topics)
	})

	// Create topic + spawn chat agent.
	g.POST("/projects/:pid/topics", func(c *gin.Context) {
		var req struct {
			Title  string `json:"title" binding:"required"`
			Plugin string `json:"plugin" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			badRequest(c, err.Error())
			return
		}

		pid := c.Param("pid")
		proj, err := d.Store.GetProject(c.Request.Context(), pid)
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
			return
		}
		if err != nil {
			internalError(c, err)
			return
		}

		// Validate plugin: must exist, be enabled, and be a chat agent.
		mem, err := d.Store.GetAgentMember(c.Request.Context(), req.Plugin)
		if errors.Is(err, store.ErrNotFound) {
			badRequest(c, "unknown plugin: "+req.Plugin)
			return
		}
		if err != nil {
			internalError(c, err)
			return
		}
		if !mem.Enabled {
			badRequest(c, "plugin is disabled: "+req.Plugin)
			return
		}
		if mem.Kind != "chat" {
			badRequest(c, "plugin is not a chat agent: "+req.Plugin)
			return
		}

		manifest, err := d.Chat.LoadManifestByName(req.Plugin)
		if err != nil {
			internalError(c, err)
			return
		}

		topicID, err := uuid.NewV7()
		if err != nil {
			internalError(c, err)
			return
		}

		topic, err := d.Store.CreateTopic(c.Request.Context(), topicID.String(), store.TopicCreate{
			ProjectID: pid,
			Title:     req.Title,
			Plugin:    req.Plugin,
		})
		if err != nil {
			internalError(c, err)
			return
		}

		agentPID, err := d.Chat.Start(c.Request.Context(), topic.ID, manifest, proj.RepoLocal, nil)
		if err != nil {
			// Spawn failed — return the topic but with an error detail.
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":    "spawn failed",
				"detail":   err.Error(),
				"topic_id": topic.ID,
			})
			return
		}
		if err := d.Store.UpdateTopicPID(c.Request.Context(), topic.ID, &agentPID); err != nil {
			internalError(c, err)
			return
		}
		topic.PID = &agentPID
		c.JSON(http.StatusCreated, topic)
	})

	// Get single topic.
	g.GET("/topics/:id", func(c *gin.Context) {
		t, err := d.Store.GetTopic(c.Request.Context(), c.Param("id"))
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if err != nil {
			internalError(c, err)
			return
		}
		c.JSON(http.StatusOK, t)
	})

	// Close topic (kill agent + mark closed).
	g.DELETE("/topics/:id", func(c *gin.Context) {
		id := c.Param("id")
		t, err := d.Store.GetTopic(c.Request.Context(), id)
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if err != nil {
			internalError(c, err)
			return
		}
		if t.Status == "closed" {
			c.Status(http.StatusNoContent)
			return
		}
		_ = d.Chat.Close(id)
		if err := d.Store.CloseTopic(c.Request.Context(), id); err != nil {
			internalError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})

	// List messages (polling target).
	g.GET("/topics/:id/messages", func(c *gin.Context) {
		after, _ := strconv.ParseInt(c.DefaultQuery("after", "0"), 10, 64)
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
		msgs, err := d.Store.ListMessages(c.Request.Context(), c.Param("id"), after, limit)
		if err != nil {
			internalError(c, err)
			return
		}
		c.JSON(http.StatusOK, msgs)
	})

	// Send user message.
	g.POST("/topics/:id/messages", func(c *gin.Context) {
		var req struct {
			Content string `json:"content" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			badRequest(c, err.Error())
			return
		}

		id := c.Param("id")
		t, err := d.Store.GetTopic(c.Request.Context(), id)
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "topic not found"})
			return
		}
		if err != nil {
			internalError(c, err)
			return
		}
		if t.Status != "open" {
			c.JSON(http.StatusConflict, gin.H{"error": "topic is closed"})
			return
		}
		if !d.Chat.IsAlive(id) {
			c.JSON(http.StatusConflict, gin.H{"error": "agent not running"})
			return
		}

		msgID, err := uuid.NewV7()
		if err != nil {
			internalError(c, err)
			return
		}

		msg, err := d.Store.CreateMessage(c.Request.Context(), msgID.String(), store.MessageCreate{
			TopicID: id,
			Role:    "user",
			Content: req.Content,
		})
		if err != nil {
			internalError(c, err)
			return
		}

		if err := d.Chat.Send(id, msg.ID, req.Content); err != nil {
			internalError(c, err)
			return
		}

		c.JSON(http.StatusCreated, msg)
	})

	// Restart dead agent for a topic.
	g.POST("/topics/:id/restart", func(c *gin.Context) {
		id := c.Param("id")
		t, err := d.Store.GetTopic(c.Request.Context(), id)
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "topic not found"})
			return
		}
		if err != nil {
			internalError(c, err)
			return
		}
		if t.Status != "open" {
			c.JSON(http.StatusConflict, gin.H{"error": "topic is closed"})
			return
		}

		// Load history for re-injection.
		history, err := d.Store.ListMessages(c.Request.Context(), id, 0, 0)
		if err != nil {
			internalError(c, err)
			return
		}

		proj, err := d.Store.GetProject(c.Request.Context(), t.ProjectID)
		if err != nil {
			internalError(c, err)
			return
		}

		manifest, err := d.Chat.LoadManifestByName(t.Plugin)
		if err != nil {
			internalError(c, err)
			return
		}

		// Close existing session if any.
		_ = d.Chat.Close(id)

		agentPID, err := d.Chat.Start(c.Request.Context(), id, manifest, proj.RepoLocal, history)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":  "restart failed",
				"detail": err.Error(),
			})
			return
		}
		if err := d.Store.UpdateTopicPID(c.Request.Context(), id, &agentPID); err != nil {
			internalError(c, err)
			return
		}
		t.PID = &agentPID
		c.JSON(http.StatusOK, t)
	})

}
