# Vibe-coded

# packwiz-pull-serve

A Go application that automatically pulls a GitHub repository on push webhooks and serves Minecraft modpack files via packwiz format. Everything is containerized in a single Docker image.

## Features

- 🚀 **GitHub Webhook Support**: Automatically triggers repository pull on push events to a specified branch
- 🔐 **HMAC-SHA256 Signature Validation**: Secure webhook verification
- 🔑 **SSH Deploy Key Authentication**: Uses GitHub deploy keys for repository access
- 📦 **Packwiz File Serving**: HTTP server for distributing modpack files
- 🐳 **Docker**: Single-image containerization with multi-stage build
- ⚙️ **Environment Configuration**: Complete configuration via environment variables
- 📊 **Health & Status Endpoints**: Built-in monitoring endpoints

## Architecture

```
┌─────────────────┐
│  GitHub Hooks   │
└────────┬────────┘
         │ (POST with signature)
         │
    ┌────▼────────────────────┐
    │  Webhook Listener       │
    │  Port: WEBHOOK_PORT     │
    │  - Validate signature   │
    │  - Extract branch name  │
    │  - Queue git pull       │
    └────┬────────────────────┘
         │
    ┌────▼────────────────────┐
    │  Git Pull Worker        │
    │  - Mutex-serialized     │
    │  - SSH authentication   │
    │  - Clone/pull via git   │
    └────┬────────────────────┘
         │
    ┌────▼────────────────────┐
    │  Repository Directory   │
    │  /tmp/packwiz-repo      │
    └─────────────────────────┘
         │
    ┌────▼────────────────────────────┐
    │  File Server (HTTP)             │
    │  Port: FILE_SERVE_PORT          │
    │  - Serves packwiz files         │
    │  - /health, /status endpoints   │
    └─────────────────────────────────┘
```

## Quick Start

### Prerequisites

- Docker and Docker Compose
- GitHub repository with a packwiz modpack
- GitHub deploy key (SSH key pair)

### 1. Create SSH Deploy Key

Generate an SSH key pair for your repository:

```bash
ssh-keygen -t ed25519 -f ./deploy-key -N ""
```

This creates `deploy-key` (private) and `deploy-key.pub` (public).

### 2. Add Deploy Key to GitHub Repository

1. Go to your repository settings: **Settings → Deploy keys**
2. Click "Add deploy key"
3. Paste the contents of `deploy-key.pub`
4. Enable "Allow write access" if you need to push back (optional)
5. Save

### 3. Base64-Encode the Deploy Key

The application expects the private key as base64. Encode it:

```bash
cat deploy-key | base64 -w0 > deploy-key.b64
```

### 4. Create GitHub Webhook Secret

Generate a random secret (used for signature validation):

```bash
openssl rand -hex 32
```

### 5. Configure and Run

Create a `.env` file:

```bash
GITHUB_WEBHOOK_SECRET=<your-generated-secret>
TARGET_REPO=github.com/your-username/your-repo
TARGET_BRANCH=main
DEPLOY_KEY=$(cat deploy-key.b64)
WEBHOOK_PORT=8080
FILE_SERVE_PORT=8081
```

Run with Docker Compose:

```bash
docker-compose up -d
```

When using Docker Compose with a key file, the default path is `./keys/deploy-key` and it is mounted as a Docker secret at `/run/secrets/deploy_key`. You can override host key file path with:

```bash
DEPLOY_KEY_FILE=./keys/deploy-key docker-compose up -d
```

### 6. Configure GitHub Webhook

1. Go to your repository: **Settings → Webhooks**
2. Click "Add webhook"
3. **Payload URL**: `http://your-server-ip:8080/webhook`
4. **Content type**: `application/json`
5. **Secret**: Paste your generated secret from step 4
6. **Events**: Select "Just the push event"
7. Save

## Environment Variables

| Variable                | Required | Default             | Description                                                   |
| ----------------------- | -------- | ------------------- | ------------------------------------------------------------- |
| `GITHUB_WEBHOOK_SECRET` | ✅       | -                   | Secret for HMAC-SHA256 signature validation                   |
| `TARGET_REPO`           | ✅       | -                   | Repository in format `github.com/owner/repo`                  |
| `TARGET_BRANCH`         | ✅       | -                   | Branch to pull on push events (e.g., `main`)                  |
| `DEPLOY_KEY`            | ⚠️\*     | -                   | Base64-encoded SSH private key **OR**                         |
| `DEPLOY_KEY_PATH`       | ⚠️\*     | -                   | Path to SSH private key file (if mounted)                     |
| `WEBHOOK_PORT`          | ❌       | `8080`              | Port for webhook listener                                     |
| `FILE_SERVE_PORT`       | ❌       | `8081`              | Port for file serving                                         |
| `LISTEN_ADDR`           | ❌       | `0.0.0.0`           | Address to bind to                                            |
| `REPO_PATH`             | ❌       | `/tmp/packwiz-repo` | Local path for repository clone                               |
| `FILE_SERVE_PATH`       | ❌       | `/`                 | Subdirectory within repo to serve (e.g., `/pack`, `/modpack`) |

_\*Either `DEPLOY_KEY` or `DEPLOY_KEY_PATH` must be provided_

## API Endpoints

### `POST /webhook`

GitHub webhook endpoint. Validates HMAC-SHA256 signature and triggers pull on matching branch.

**Response**: `202 Accepted` (on success), `200 OK` (wrong branch), `401 Unauthorized` (bad signature)

### `GET /health`

Health check endpoint.

**Response**: `200 OK` with "OK"

### `GET /status`

Current status including last pull timestamp and errors.

**Response**:

```json
{
  "status": "ok",
  "last_pull": "2025-04-24T10:15:30Z",
  "last_error": ""
}
```

### `GET /` (and subpaths)

File serving from the configured `FILE_SERVE_PATH`. Serves packwiz files with proper caching headers.

## Docker Build & Run

### Build the image:

```bash
docker build -t packwiz-pull-serve:latest .
```

### Run directly:

```bash
docker run -d \
  -p 8080:8080 \
  -p 8081:8081 \
  -e GITHUB_WEBHOOK_SECRET="your-secret" \
  -e TARGET_REPO="github.com/owner/repo" \
  -e TARGET_BRANCH="main" \
  -e DEPLOY_KEY="your-base64-key" \
  -v packwiz-repo:/tmp/packwiz-repo \
  --name packwiz-pull-serve \
  packwiz-pull-serve:latest
```

### Run with docker-compose:

```bash
docker-compose up -d
```

### View logs:

```bash
docker-compose logs -f packwiz-pull-serve
```

## Configuration Examples

### Example 1: Basic Minecraft Modpack

```env
GITHUB_WEBHOOK_SECRET=abc123def456...
TARGET_REPO=github.com/myteam/minecraft-pack
TARGET_BRANCH=main
DEPLOY_KEY=LS0tLS1CRUdJTi... (base64-encoded key)
WEBHOOK_PORT=8080
FILE_SERVE_PORT=8081
FILE_SERVE_PATH=/
```

### Example 2: Modpack in Subdirectory

```env
GITHUB_WEBHOOK_SECRET=abc123def456...
TARGET_REPO=github.com/myteam/monorepo
TARGET_BRANCH=release
DEPLOY_KEY=LS0tLS1CRUdJTi... (base64-encoded key)
WEBHOOK_PORT=8080
FILE_SERVE_PORT=8081
FILE_SERVE_PATH=/modpacks/my-pack
```

### Example 3: Multiple Instances (Different Branches)

**docker-compose-dev.yml** (development branch):

```yaml
services:
  packwiz-dev:
    # ... (same as main, but different ports)
    ports:
      - "8082:8080"
      - "8083:8081"
    environment:
      TARGET_BRANCH: develop
```

**docker-compose-prod.yml** (production branch):

```yaml
services:
  packwiz-prod:
    # ... (same as main)
    ports:
      - "8080:8080"
      - "8081:8081"
    environment:
      TARGET_BRANCH: main
```

## Troubleshooting

### Webhook not triggering pulls

1. Check webhook logs in GitHub repository settings
2. Verify secret matches in both GitHub and application
3. Confirm target branch is correct
4. Check application logs: `docker-compose logs -f packwiz-pull-serve`

### "Permission denied" or "Could not read SSH key"

1. Verify deploy key is base64-encoded correctly
2. Ensure public key is added to GitHub repository
3. Check key format (should be valid SSH private key)
4. Verify network access to github.com:22

### File serving returns 404

1. Verify `FILE_SERVE_PATH` is correct
2. Check that files are present in repository
3. Ensure repository was cloned successfully: `docker exec <container> ls /tmp/packwiz-repo`
4. Check file permissions

### Application won't start

1. Review logs: `docker-compose logs packwiz-pull-serve`
2. Verify all required environment variables are set
3. Ensure ports are not already in use: `netstat -tuln | grep 8080`
4. Check disk space for repository clone

## Development

### Project Structure

```
.
├── main.go                      # Application entry point
├── go.mod                       # Go module definition
├── Dockerfile                   # Multi-stage Docker build
├── docker-compose.yml           # Docker Compose configuration
├── README.md                    # This file
├── .dockerignore               # Docker build exclusions
├── .gitignore                  # Git exclusions
└── internal/
    ├── config/
    │   └── config.go           # Environment variable loading
    ├── webhook/
    │   ├── handler.go          # Webhook HTTP handler
    │   └── parser.go           # GitHub event parsing
    ├── git/
    │   └── manager.go          # Git operations (clone/pull/SSH)
    └── server/
        └── server.go           # HTTP server and routes
```

### Building Locally

```bash
go mod download
go build -o packwiz-pull-serve
./packwiz-pull-serve
```

Set environment variables before running, or create a `.env` file and load it.

## Security Considerations

1. **Webhook Secret**: Use a strong, random secret (minimum 32 characters)
2. **SSH Keys**: Never commit private keys to version control
3. **Deploy Keys**: Create repository-specific deploy keys, not personal SSH keys
4. **HTTPS**: Use a reverse proxy (nginx, traefik) for HTTPS in production
5. **Network**: Restrict webhook endpoint to GitHub IP ranges in production
6. **File Permissions**: Application runs as non-root user in Docker

## Performance Notes

- **Git Operations**: Serialized with `sync.RWMutex` to prevent concurrent writes
- **File Serving**: Concurrent, no locking required
- **Channel Buffering**: Webhook handler queues operations with buffer size 1 (prevents duplicate concurrent pulls)
- **SSH Caching**: No connection pooling; each pull establishes new SSH connection

## License

Open source - modify and use freely.

## Contributing

Contributions welcome! Please submit issues and pull requests.
