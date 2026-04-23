package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	// GitHub Configuration
	GitHubWebhookSecret string
	TargetBranch        string
	TargetRepo          string // e.g., github.com/owner/repo
	DeployKeyPath       string // Path to SSH private key file
	DeployKeyBase64     string // Alternative: base64-encoded key (written to temp file)

	// Repository Configuration
	RepoPath      string // Local path where repo will be cloned
	FileServePath string // Subdirectory within repo to serve (e.g., "/" or "/pack")

	// Server Configuration
	WebhookPort   int
	FileServePort int
	ListenAddr    string // Address to bind to (default "0.0.0.0")
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		GitHubWebhookSecret: os.Getenv("GITHUB_WEBHOOK_SECRET"),
		TargetBranch:        os.Getenv("TARGET_BRANCH"),
		TargetRepo:          os.Getenv("TARGET_REPO"),
		DeployKeyPath:       os.Getenv("DEPLOY_KEY_PATH"),
		DeployKeyBase64:     os.Getenv("DEPLOY_KEY"),
		RepoPath:            os.Getenv("REPO_PATH"),
		FileServePath:       os.Getenv("FILE_SERVE_PATH"),
		ListenAddr:          os.Getenv("LISTEN_ADDR"),
	}

	// Set defaults
	if cfg.RepoPath == "" {
		cfg.RepoPath = "/tmp/packwiz-repo"
	}
	if cfg.FileServePath == "" {
		cfg.FileServePath = "/"
	}
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = "0.0.0.0"
	}

	// Parse ports
	webhookPortStr := os.Getenv("WEBHOOK_PORT")
	if webhookPortStr == "" {
		webhookPortStr = "8080"
	}
	webhookPort, err := strconv.Atoi(webhookPortStr)
	if err != nil {
		return nil, fmt.Errorf("invalid WEBHOOK_PORT: %w", err)
	}
	cfg.WebhookPort = webhookPort

	fileServePortStr := os.Getenv("FILE_SERVE_PORT")
	if fileServePortStr == "" {
		fileServePortStr = "8081"
	}
	fileServePort, err := strconv.Atoi(fileServePortStr)
	if err != nil {
		return nil, fmt.Errorf("invalid FILE_SERVE_PORT: %w", err)
	}
	cfg.FileServePort = fileServePort

	// Validate required fields
	if cfg.GitHubWebhookSecret == "" {
		return nil, fmt.Errorf("GITHUB_WEBHOOK_SECRET is required")
	}
	if cfg.TargetBranch == "" {
		return nil, fmt.Errorf("TARGET_BRANCH is required")
	}
	if cfg.TargetRepo == "" {
		return nil, fmt.Errorf("TARGET_REPO is required")
	}
	if cfg.DeployKeyPath == "" && cfg.DeployKeyBase64 == "" {
		return nil, fmt.Errorf("either DEPLOY_KEY_PATH or DEPLOY_KEY (base64) is required")
	}

	// If base64 key is provided, decode and write to temp file
	if cfg.DeployKeyBase64 != "" && cfg.DeployKeyPath == "" {
		decoded, err := base64.StdEncoding.DecodeString(cfg.DeployKeyBase64)
		if err != nil {
			return nil, fmt.Errorf("failed to decode DEPLOY_KEY base64: %w", err)
		}

		// Write to temp file
		keyFile, err := os.CreateTemp("", "deploy-key-*")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp key file: %w", err)
		}
		defer keyFile.Close()

		if _, err := keyFile.Write(decoded); err != nil {
			return nil, fmt.Errorf("failed to write deploy key: %w", err)
		}

		// Set restrictive permissions (SSH requires this)
		if err := os.Chmod(keyFile.Name(), 0600); err != nil {
			return nil, fmt.Errorf("failed to set key permissions: %w", err)
		}

		cfg.DeployKeyPath = keyFile.Name()
	}

	// Ensure file serve path starts with /
	if !strings.HasPrefix(cfg.FileServePath, "/") {
		cfg.FileServePath = "/" + cfg.FileServePath
	}

	return cfg, nil
}
