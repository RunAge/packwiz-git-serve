package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

// Handler processes GitHub webhook events
type Handler struct {
	secret       string
	targetBranch string
	gitPullChan  chan<- struct{}
	logger       *slog.Logger
}

// NewHandler creates a new webhook handler
func NewHandler(secret, targetBranch string, gitPullChan chan<- struct{}, logger *slog.Logger) *Handler {
	return &Handler{
		secret:       secret,
		targetBranch: targetBranch,
		gitPullChan:  gitPullChan,
		logger:       logger,
	}
}

// ServeHTTP handles incoming webhook requests
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.logger.Warn("webhook endpoint received non-POST request", "method", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("failed to read request body", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Validate signature
	signature := r.Header.Get("X-Hub-Signature-256")
	if !h.validateSignature(body, signature) {
		h.logger.Warn("webhook signature validation failed")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Unauthorized")
		return
	}

	// Parse payload
	var payload PushEvent
	if err := json.Unmarshal(body, &payload); err != nil {
		h.logger.Error("failed to parse webhook payload", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Extract branch from ref (format: "refs/heads/branch-name")
	branch := extractBranchFromRef(payload.Ref)

	h.logger.Info("webhook received", "branch", branch, "ref", payload.Ref)

	// Check if branch matches target
	if branch != h.targetBranch {
		h.logger.Info("push to non-target branch, ignoring", "branch", branch, "target", h.targetBranch)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Push to branch %s, target is %s\n", branch, h.targetBranch)
		return
	}

	// Queue git pull operation
	h.logger.Info("queuing git pull operation", "branch", branch)
	select {
	case h.gitPullChan <- struct{}{}:
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprint(w, "Pull operation queued")
	default:
		h.logger.Warn("git pull queue full, cannot queue operation")
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, "Pull queue full")
	}
}

// validateSignature validates GitHub webhook signature using HMAC-SHA256
func (h *Handler) validateSignature(body []byte, signature string) bool {
	if signature == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(h.secret))
	mac.Write(body)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	// Use constant-time comparison to prevent timing attacks
	return hmac.Equal([]byte(expected), []byte(signature))
}

// extractBranchFromRef extracts branch name from GitHub ref (e.g., "refs/heads/main" -> "main")
func extractBranchFromRef(ref string) string {
	if len(ref) > 11 && ref[:11] == "refs/heads/" {
		return ref[11:]
	}
	return ref
}
