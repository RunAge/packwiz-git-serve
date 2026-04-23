package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/runage/packwiz-pull-serve/internal/git"
)

// Server handles HTTP requests for webhooks and file serving
type Server struct {
	webhookHandler http.Handler
	fileServePath  string
	repoPath       string
	git            *git.Manager
	logger         *slog.Logger
}

// NewServer creates a new HTTP server instance
func NewServer(webhookHandler http.Handler, fileServePath, repoPath string, gitManager *git.Manager, logger *slog.Logger) *Server {
	return &Server{
		webhookHandler: webhookHandler,
		fileServePath:  fileServePath,
		repoPath:       repoPath,
		git:            gitManager,
		logger:         logger,
	}
}

// RegisterRoutes registers all HTTP routes
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	// Webhook endpoint
	mux.Handle("/webhook", s.webhookHandler)

	// Health check endpoint
	mux.HandleFunc("/health", s.handleHealth)

	// Status endpoint
	mux.HandleFunc("/status", s.handleStatus)

	// File serving
	mux.HandleFunc("/", s.handleFileServe)
}

// handleHealth returns a 200 OK for health checks
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "OK")
}

// handleStatus returns current status information
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	pullTime, lastErr := s.git.Status()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	status := "ok"
	errMsg := ""
	if lastErr != nil {
		status = "error"
		errMsg = lastErr.Error()
	}

	lastPullStr := "never"
	if !pullTime.IsZero() {
		lastPullStr = pullTime.Format(time.RFC3339)
	}

	fmt.Fprintf(w, `{"status":"%s","last_pull":"%s","last_error":"%s"}`+"\n", status, lastPullStr, errMsg)
}

// handleFileServe serves files from the repository
func (s *Server) handleFileServe(w http.ResponseWriter, r *http.Request) {
	// Construct the full path to serve
	servePath := filepath.Join(s.repoPath, s.fileServePath, r.URL.Path)

	// Security: prevent path traversal
	realPath, err := filepath.EvalSymlinks(servePath)
	if err != nil {
		s.logger.Warn("failed to resolve path", "path", servePath, "error", err)
		http.NotFound(w, r)
		return
	}

	serveDir := filepath.Join(s.repoPath, s.fileServePath)
	if !isPathWithin(realPath, serveDir) {
		s.logger.Warn("attempted path traversal", "path", servePath)
		http.NotFound(w, r)
		return
	}

	// Serve the file
	fs := http.FileServer(http.Dir(serveDir))
	fs.ServeHTTP(w, r)
}

// isPathWithin checks if a path is within a base directory
func isPathWithin(path, base string) bool {
	absBase, err := filepath.Abs(base)
	if err != nil {
		return false
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	// Ensure both paths end consistently for comparison
	rel, err := filepath.Rel(absBase, absPath)
	if err != nil {
		return false
	}

	// Keep paths inside base directory, including the base directory itself.
	if rel == "." {
		return true
	}
	if rel == ".." {
		return false
	}

	parentPrefix := ".." + string(filepath.Separator)
	return !strings.HasPrefix(rel, parentPrefix)
}
