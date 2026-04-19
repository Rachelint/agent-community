package api

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

func badRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, gin.H{"error": msg})
}

func internalError(c *gin.Context, err error) {
	slog.Error("internal error", "path", c.FullPath(), "err", err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
}
