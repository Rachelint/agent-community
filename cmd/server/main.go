package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Rachelint/agent-community/internal/api"
	"github.com/Rachelint/agent-community/internal/config"
	"github.com/Rachelint/agent-community/internal/plugin"
	"github.com/Rachelint/agent-community/internal/reconcile"
	"github.com/Rachelint/agent-community/internal/store"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := config.Load()

	ctx, cancelStore := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelStore()
	dbPath := filepath.Join(cfg.DataDir, "db.sqlite")
	st, err := store.Open(ctx, dbPath)
	if err != nil {
		slog.Error("open store", "err", err, "path", dbPath)
		os.Exit(1)
	}
	defer func() { _ = st.Close() }()
	slog.Info("store ready", "path", dbPath)

	pm, err := plugin.NewManager(
		plugin.DataDirs{
			RepoAgentsDir: cfg.RepoAgentsDir,
			UserAgentsDir: cfg.UserAgentsDir,
		},
		cfg.CallbackBaseURL,
	)
	if err != nil {
		slog.Error("init plugin manager", "err", err)
		os.Exit(1)
	}
	slog.Info("plugin manager ready",
		"repo_agents", cfg.RepoAgentsDir,
		"user_agents", cfg.UserAgentsDir,
		"callback", cfg.CallbackBaseURL)

	// Workspace root for spawned runs.
	workspacesDir := filepath.Join(cfg.DataDir, "workspaces")
	if err := os.MkdirAll(workspacesDir, 0o755); err != nil {
		slog.Error("create workspaces dir", "err", err)
		os.Exit(1)
	}

	// Chat manager for long-running chat agent processes.
	chatMgr := plugin.NewChatManager(st, plugin.DataDirs{
		RepoAgentsDir: cfg.RepoAgentsDir,
		UserAgentsDir: cfg.UserAgentsDir,
	}, cfg.DataDir)
	if err := chatMgr.RecoverOnStartup(ctx); err != nil {
		slog.Warn("chat recover on startup", "err", err)
	}

	// Start the reconciler that polls for done.json / dead processes.
	rec := reconcile.New(st, pm, 5*time.Second)
	rec.Run()
	defer rec.Stop()

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// CORS: allow any origin during development.
	r.Use(func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
			c.Header("Access-Control-Max-Age", "86400")
		}
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	api.Register(r, api.Deps{
		Store:         st,
		Plugin:        pm,
		Chat:          chatMgr,
		WorkspacesDir: workspacesDir,
	})

	serveUI(r)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("server listening", "addr", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	slog.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
}
