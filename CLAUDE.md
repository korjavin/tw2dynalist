# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

This is a Twitter to Dynalist bot written in Go that monitors Twitter bookmarks and automatically adds them to a Dynalist inbox. The application uses OAuth 2.0 for Twitter authentication and runs as a containerized service with a web dashboard.

## Architecture

The codebase follows a clean architecture pattern with clear separation of concerns:

- **`main.go`**: Entry point that initializes and runs the application
- **`internal/app/`**: Core application logic, dependency injection, HTTP handlers, and metrics
- **`internal/config/`**: Configuration management using environment variables
- **`internal/auth/`**: OAuth 2.0 authentication handling for Twitter
- **`internal/twitter/`**: Twitter API client for fetching and managing bookmarks
- **`internal/dynalist/`**: Dynalist API client for adding items to inbox
- **`internal/storage/`**: Local file-based storage for caching processed tweets
- **`internal/scheduler/`**: Periodic task scheduler for bookmark processing
- **`internal/logger/`**: Structured logging with configurable levels

The application runs a web server on port 8080 (configurable) that serves:
- `/`: Dashboard with status and metrics
- `/callback`: OAuth 2.0 callback endpoint for Twitter authentication
- `/api/metrics`: JSON metrics endpoint

## Common Commands

### Building and Testing
```bash
# Run all tests
go test ./...

# Build binary locally
go build -o tw2dynalist

# Build Docker image locally
./scripts/local-build.sh
# or manually: docker build -t tw2dynalist:local .
```

### Development and Running
```bash
# Run with Go (requires environment variables)
go run .

# Run with Docker Compose (recommended for development)
docker-compose up -d

# View logs
docker-compose logs -f tw2dynalist

# Run locally built image
docker run --rm -it --env-file .env -p 8080:8080 -v $(pwd)/data:/app/data tw2dynalist:local
```

### Environment Setup
Copy `.env.example` to `.env` and configure required variables:
- `DYNALIST_TOKEN`: Dynalist API token
- `TWITTER_CLIENT_ID`: Twitter OAuth 2.0 Client ID  
- `TWITTER_CLIENT_SECRET`: Twitter OAuth 2.0 Client Secret
- `TW_USER`: Twitter username to monitor
- `TWITTER_REDIRECT_URL`: OAuth callback URL (default: http://localhost:8080/callback)

## OAuth 2.0 Flow

The application uses OAuth 2.0 with PKCE for Twitter authentication:
1. On first run, it starts a callback server and displays an authorization URL
2. User visits URL in browser to authorize the application
3. Twitter redirects to callback server with authorization code
4. Application exchanges code for access token and stores in `token.json`
5. Subsequent runs use stored token with automatic refresh

## Key Configuration Options

- `CHECK_INTERVAL`: How often to check for new bookmarks (default: 1h)
- `LOG_LEVEL`: Logging verbosity (DEBUG, INFO, WARN, ERROR)
- `REMOVE_BOOKMARKS`: Remove bookmarks from Twitter after processing (default: false)
- `CLEANUP_PROCESSED_BOOKMARKS`: One-time cleanup of already processed bookmarks (default: false)
- `CALLBACK_PORT`: Port for OAuth callback server (default: 8080)

## Docker Deployment

The application is designed for containerized deployment:
- Multi-architecture Docker images (amd64/arm64) published to `ghcr.io/korjavin/tw2dynalist`
- Includes health checks and volume mounts for persistent data
- Supports Traefik integration for domain-based OAuth callbacks
- CI/CD pipeline with automated builds and Portainer webhook deployment

## Testing Strategy

Tests are organized per package with `*_test.go` files. The CI pipeline runs `go test ./...` to execute all tests before building Docker images.