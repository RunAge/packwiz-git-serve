package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/runage/packwiz-pull-serve/internal/config"
	"github.com/runage/packwiz-pull-serve/internal/git"
	"github.com/runage/packwiz-pull-serve/internal/server"
	"github.com/runage/packwiz-pull-serve/internal/webhook"
)

func main() {
	// Setup logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	logger.Info("configuration loaded", "repo", cfg.TargetRepo, "branch", cfg.TargetBranch, "webhook_port", cfg.WebhookPort, "file_serve_port", cfg.FileServePort)

	// Ensure repo path exists
	if err := os.MkdirAll(cfg.RepoPath, 0755); err != nil {
		logger.Error("failed to create repo path", "path", cfg.RepoPath, "error", err)
		os.Exit(1)
	}

	// Initialize git manager
	gitManager := git.NewManager(
		toGitSSHURL(cfg.TargetRepo),
		cfg.RepoPath,
		cfg.TargetBranch,
		cfg.DeployKeyPath,
		logger,
	)

	// Perform initial pull
	logger.Info("performing initial repository pull...")
	if err := gitManager.Pull(); err != nil {
		logger.Warn("initial pull failed (may retry on webhook)", "error", err)
	} else {
		logger.Info("initial pull successful")
	}

	// Create a channel for git operations
	gitPullChan := make(chan struct{}, 1)

	// Start git pull worker
	go gitPullWorker(gitPullChan, gitManager, logger)

	// Create webhook handler
	webhookHandler := webhook.NewHandler(cfg.GitHubWebhookSecret, cfg.TargetBranch, gitPullChan, logger)

	// Create HTTP server
	serverInstance := server.NewServer(webhookHandler, cfg.FileServePath, cfg.RepoPath, gitManager, logger)

	// Setup routes
	mux := http.NewServeMux()
	serverInstance.RegisterRoutes(mux)

	// Start webhook server
	webhookServer := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.ListenAddr, cfg.WebhookPort),
		Handler: mux,
	}

	// Also start file serve server if different port
	var fileServeServer *http.Server
	if cfg.WebhookPort != cfg.FileServePort {
		fileServeServer = &http.Server{
			Addr:    fmt.Sprintf("%s:%d", cfg.ListenAddr, cfg.FileServePort),
			Handler: mux, // Same routes, can access files on this port too
		}
	}

	// Graceful shutdown handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start webhook server
	go func() {
		logger.Info("starting webhook server", "address", webhookServer.Addr)
		if err := webhookServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("webhook server error", "error", err)
		}
	}()

	// Start file serve server if different port
	if fileServeServer != nil {
		go func() {
			logger.Info("starting file serve server", "address", fileServeServer.Addr)
			if err := fileServeServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error("file serve server error", "error", err)
			}
		}()
	}

	// Wait for shutdown signal
	<-sigChan
	logger.Info("shutdown signal received")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	webhookServer.Shutdown(ctx)
	if fileServeServer != nil {
		fileServeServer.Shutdown(ctx)
	}

	logger.Info("server shutdown complete")
}

func toGitSSHURL(targetRepo string) string {
	repo := strings.TrimSpace(targetRepo)

	if strings.HasPrefix(repo, "git@") || strings.HasPrefix(repo, "ssh://") || strings.HasPrefix(repo, "https://") || strings.HasPrefix(repo, "http://") {
		return repo
	}

	repo = strings.TrimPrefix(repo, "github.com/")
	repo = strings.TrimPrefix(repo, "github.com:")
	repo = strings.TrimSuffix(repo, ".git")

	return fmt.Sprintf("git@github.com:%s.git", repo)
}

// gitPullWorker processes git pull operations from the channel
func gitPullWorker(pullChan <-chan struct{}, gitManager *git.Manager, logger *slog.Logger) {
	for range pullChan {
		logger.Info("processing queued git pull")
		if err := gitManager.Pull(); err != nil {
			logger.Error("git pull operation failed", "error", err)
		} else {
			logger.Info("git pull operation completed successfully")
		}
	}
}
