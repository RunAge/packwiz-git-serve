# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git make

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o packwiz-pull-serve .

# Runtime stage
FROM alpine:latest

# Install runtime dependencies (git, openssh for SSH support)
RUN apk add --no-cache git openssh-client ca-certificates

# Create app user
RUN addgroup -g 1000 app && adduser -D -u 1000 -G app app

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/packwiz-pull-serve /app/packwiz-pull-serve

# Create SSH directory with proper permissions
RUN mkdir -p /home/app/.ssh && chown -R app:app /home/app

# Change to app user
USER app

# Expose ports (webhook and file serve)
# Can be overridden at runtime
EXPOSE 8080 8081

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
ENTRYPOINT ["/app/packwiz-pull-serve"]
