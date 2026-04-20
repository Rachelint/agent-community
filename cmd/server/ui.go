package main

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed all:ui
var uiFS embed.FS

func serveUI(r *gin.Engine) {
	sub, err := fs.Sub(uiFS, "ui")
	if err != nil {
		return
	}
	fileServer := http.FileServer(http.FS(sub))

	r.NoRoute(func(c *gin.Context) {
		// Try to serve the file directly.
		path := c.Request.URL.Path
		if len(path) > 1 {
			if f, err := sub.Open(path[1:]); err == nil {
				f.Close()
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
		}
		// SPA fallback: serve index.html for any non-API route.
		c.FileFromFS("index.html", http.FS(sub))
	})
}
