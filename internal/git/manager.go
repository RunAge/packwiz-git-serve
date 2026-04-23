package git

import (
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	gssh "golang.org/x/crypto/ssh"
)

// Manager handles git operations with SSH authentication and mutex-based serialization
type Manager struct {
	repoURL       string
	repoPath      string
	targetBranch  string
	deployKeyPath string
	logger        *slog.Logger
	mu            sync.RWMutex
	lastPullTime  time.Time
	lastError     error
}

// NewManager creates a new git manager
func NewManager(repoURL, repoPath, targetBranch, deployKeyPath string, logger *slog.Logger) *Manager {
	return &Manager{
		repoURL:       repoURL,
		repoPath:      repoPath,
		targetBranch:  targetBranch,
		deployKeyPath: deployKeyPath,
		logger:        logger,
	}
}

// Pull performs a git pull operation (clone if not exists, otherwise fetch and pull)
func (m *Manager) Pull() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Info("starting git pull operation", "repo", m.repoURL, "branch", m.targetBranch)

	// Create SSH public key auth
	auth, err := m.createSSHAuth()
	if err != nil {
		m.lastError = err
		m.logger.Error("failed to create SSH auth", "error", err)
		return err
	}

	// Clone when repo path does not exist or when it exists but is not a git repo yet.
	if _, err := os.Stat(m.repoPath); os.IsNotExist(err) {
		return m.clone(auth)
	}
	if _, err := os.Stat(m.repoPath + "/.git"); os.IsNotExist(err) {
		return m.clone(auth)
	}

	// Pull existing repository
	return m.pullExisting(auth)
}

// clone clones the repository for the first time
func (m *Manager) clone(auth *ssh.PublicKeys) error {
	m.logger.Info("cloning repository", "url", m.repoURL, "path", m.repoPath)

	_, err := git.PlainClone(m.repoPath, false, &git.CloneOptions{
		URL:           m.repoURL,
		Auth:          auth,
		Depth:         0,
		Progress:      os.Stdout,
		ReferenceName: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", m.targetBranch)),
		SingleBranch:  false,
	})
	if err != nil {
		m.logger.Error("failed to clone repository", "error", err)
		m.lastError = err
		return err
	}

	m.logger.Info("repository cloned successfully")
	m.lastPullTime = time.Now()
	m.lastError = nil
	return nil
}

// pullExisting pulls the latest changes from an existing repository
func (m *Manager) pullExisting(auth *ssh.PublicKeys) error {
	m.logger.Info("pulling latest changes", "path", m.repoPath, "branch", m.targetBranch)

	repo, err := git.PlainOpen(m.repoPath)
	if err != nil {
		m.logger.Error("failed to open repository", "error", err)
		m.lastError = err
		return err
	}

	worktree, err := repo.Worktree()
	if err != nil {
		m.logger.Error("failed to get worktree", "error", err)
		m.lastError = err
		return err
	}

	// Pull from origin
	err = worktree.Pull(&git.PullOptions{
		Auth:          auth,
		Progress:      os.Stdout,
		ReferenceName: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", m.targetBranch)),
		Force:         false,
	})

	if err != nil && err != git.NoErrAlreadyUpToDate {
		m.logger.Error("failed to pull repository", "error", err)
		m.lastError = err
		return err
	}

	if err == git.NoErrAlreadyUpToDate {
		m.logger.Info("repository already up to date")
	} else {
		m.logger.Info("repository pulled successfully")
	}

	m.lastPullTime = time.Now()
	m.lastError = nil
	return nil
}

// createSSHAuth creates SSH authentication from the deploy key file
func (m *Manager) createSSHAuth() (*ssh.PublicKeys, error) {
	publicKeys, err := ssh.NewPublicKeysFromFile("git", m.deployKeyPath, "")
	if err != nil {
		return nil, fmt.Errorf("failed to load SSH key: %w", err)
	}

	// Don't verify host keys (accept all)
	publicKeys.HostKeyCallback = gssh.InsecureIgnoreHostKey()

	return publicKeys, nil
}

// GetLastPullTime returns the timestamp of the last successful pull
func (m *Manager) GetLastPullTime() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastPullTime
}

// GetLastError returns the last error that occurred
func (m *Manager) GetLastError() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastError
}

// Status returns the current status
func (m *Manager) Status() (pullTime time.Time, lastErr error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastPullTime, m.lastError
}
