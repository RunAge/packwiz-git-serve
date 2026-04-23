package webhook

// PushEvent represents a GitHub push webhook event
type PushEvent struct {
	Ref        string     `json:"ref"`
	Before     string     `json:"before"`
	After      string     `json:"after"`
	Repository Repository `json:"repository"`
	Pusher     User       `json:"pusher"`
	Commits    []Commit   `json:"commits"`
}

// Repository represents a GitHub repository in webhook payload
type Repository struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Private  bool   `json:"private"`
	HTMLURL  string `json:"html_url"`
	CloneURL string `json:"clone_url"`
	SSHURL   string `json:"ssh_url"`
}

// User represents a GitHub user/pusher in webhook payload
type User struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

// Commit represents a commit in a push event
type Commit struct {
	ID        string   `json:"id"`
	TreeID    string   `json:"tree_id"`
	Message   string   `json:"message"`
	Timestamp string   `json:"timestamp"`
	Author    User     `json:"author"`
	Committer User     `json:"committer"`
	Added     []string `json:"added"`
	Removed   []string `json:"removed"`
	Modified  []string `json:"modified"`
}
