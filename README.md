# Twitter to Dynalist Bot

This bot monitors a Twitter user's bookmarks and automatically adds them to your Dynalist inbox.

## Features

- Monitors a specified Twitter user's bookmarks
- Adds new bookmarked tweets to Dynalist inbox
- Uses local cache to avoid duplicates
- Checks for new bookmarks hourly (configurable)
- Runs in a Docker container

## Prerequisites

- Twitter OAuth 2.0 app credentials (Client ID, Client Secret)
- Dynalist API token
- Docker (for containerized deployment)

## Environment Variables

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `DYNALIST_TOKEN` | Your Dynalist API token | Yes | - |
| `TWITTER_CLIENT_ID` | Twitter OAuth 2.0 Client ID | Yes | - |
| `TWITTER_CLIENT_SECRET` | Twitter OAuth 2.0 Client Secret | Yes | - |
| `TWITTER_REDIRECT_URL` | OAuth callback URL (e.g., http://localhost:8080/callback) | Yes | - |
| `TW_USER` | Twitter username to monitor | Yes | - |
| `CACHE_FILE_PATH` | Path to cache file | No | `cache.json` |
| `TOKEN_FILE_PATH` | Path to OAuth token storage file | No | `token.json` |
| `CHECK_INTERVAL` | Interval to check for new bookmarks | No | `1h` |
| `LOG_LEVEL` | Logging level (DEBUG, INFO, WARN, ERROR) | No | `INFO` |
| `REMOVE_BOOKMARKS` | Remove bookmarks after saving to Dynalist | No | `false` |
| `CLEANUP_PROCESSED_BOOKMARKS` | One-time cleanup of already processed bookmarks | No | `false` |
| `NTFY_SERVER` | URL of the ntfy server | No | `http://ntfy:80` |
| `NTFY_TOPIC` | ntfy topic to send notifications to | No | `tw2dynalist` |
| `NTFY_PORT` | Port to expose the ntfy web UI on | No | `8081` |

## Getting Twitter API Credentials

1. Create a Twitter Developer account:
   - Go to [developer.twitter.com](https://developer.twitter.com/)
   - Sign in with your Twitter account
   - Apply for a developer account by following the on-screen instructions
   - You'll need to provide some information about how you plan to use the API

2. Create a new project and app:
   - Once your developer account is approved, go to the [Developer Portal](https://developer.twitter.com/en/portal/dashboard)
   - Create a new Project and give it a name (e.g., "Dynalist Bot")
   - Create a new App within the project

3. Set up app permissions:
   - In your app settings, navigate to the "App permissions" section
   - Change the app permissions to "Read and Write"
   - Save your changes

4. Configure OAuth 2.0 settings:
   - In your app settings, navigate to the "User authentication settings" section
   - Enable OAuth 2.0
   - Set the callback URL to `http://localhost:8080/callback` (or your preferred port)
   - Select the required scopes: `tweet.read`, `users.read`, `bookmark.read`, `bookmark.write`, `offline.access`
   - **Note**: `bookmark.write` is required only if you plan to use bookmark removal features
   - **Note**: `offline.access` is required for automatic token refresh to enable unattended operation
   - Save your changes

5. Get your OAuth 2.0 credentials:
   - In your app settings, navigate to the "Keys and tokens" tab
   - You'll find your Client ID and Client Secret under "OAuth 2.0 Client ID and Client Secret"
   - Make sure to save these values as they will be needed to configure the bot

## OAuth 2.0 Authorization Flow

This application uses OAuth 2.0 with PKCE (Proof Key for Code Exchange) for secure authentication:

1. **First Run**: When you run the app for the first time, it will:
   - Start a local callback server (default: http://localhost:8080/callback)
   - Generate a secure authorization URL
   - Display the URL for you to visit in your browser

2. **Browser Authorization**: 
   - Visit the provided URL in your browser
   - Authorize the application with your Twitter account
   - Twitter will redirect back to the callback server automatically

3. **Token Storage**: 
   - The app receives the authorization code via the callback
   - Exchanges it for an access token automatically
   - Stores the token and user info in `token.json` for future use

4. **Subsequent Runs**: 
   - Uses the stored token without requiring re-authorization
   - Automatically refreshes tokens when needed (requires `offline.access` scope)
   - Refresh tokens are valid for 6 months, providing long-term unattended operation

## Additional Requirements

- Ensure your Twitter account has the necessary permissions to access the API
- For production use, you may need to apply for Elevated access to the Twitter API
- The app uses Twitter API v2 endpoints with OAuth 2.0 User Context authentication

## Getting Dynalist API Token

1. Log in to your Dynalist account
2. Go to Settings > Developer
3. Copy your API token

## Running with Docker Compose (Recommended)

1. Clone the repository:
```bash
git clone https://github.com/korjavin/tw2dynalist.git
cd tw2dynalist
```

2. Create environment file:
```bash
cp .env.example .env
```

3. Edit `.env` file with your credentials:
```bash
# Required
DYNALIST_TOKEN=your_dynalist_api_token_here
TWITTER_CLIENT_ID=your_twitter_client_id_here
TWITTER_CLIENT_SECRET=your_twitter_client_secret_here
TW_USER=your_twitter_username_here

# Optional
TWITTER_REDIRECT_URL=http://localhost:8080/callback
LOG_LEVEL=INFO
CHECK_INTERVAL=1h
CALLBACK_PORT=8080
```

4. Start the service:
```bash
docker-compose up -d
```

5. **First-time OAuth setup**: 
   - Check logs: `docker-compose logs -f tw2dynalist`
   - Visit the OAuth URL shown in the logs to authorize the app
   - The app will automatically receive the callback and start running

6. Manage the service:
```bash
# View logs
docker-compose logs -f tw2dynalist

# Stop service
docker-compose down

# Restart service
docker-compose restart tw2dynalist

# Update to latest version
docker-compose pull && docker-compose up -d
```

## Domain-based Deployment with Traefik

For production deployments with a custom domain and HTTPS:

1. **Configure environment variables**:
```bash
# Enable Traefik integration
TRAEFIK_ENABLE=true
TW2DYNALIST_DOMAIN=tw2dynalist.yourdomain.com

# Update callback URL to use your domain
TWITTER_REDIRECT_URL=https://tw2dynalist.yourdomain.com/callback
```

2. **Update Twitter App Settings**:
   - In your Twitter Developer Portal, update the OAuth callback URL to: `https://tw2dynalist.yourdomain.com/callback`

3. **Deploy with Traefik**:
   - Ensure you have Traefik running with SSL certificate resolver
   - The compose file includes labels for automatic HTTPS setup
   - No need to expose ports when using Traefik

**Note**: When using Traefik, the callback server will be accessible via HTTPS on your domain, providing secure OAuth authentication.

## Push Notifications with ntfy

This application can send push notifications when a new bookmark is saved to Dynalist. It uses a self-hosted [ntfy](httpshttps://ntfy.sh/) service, which is included in the `docker-compose.yml` file and will be started automatically.

To receive notifications:

1.  **Start the services**: `docker-compose up -d`
2.  **Find your server IP address**. This will be the IP address of the machine running Docker.
3.  **Subscribe to the topic**:
    - **Mobile App**: Download the ntfy app for [Android](https://play.google.com/store/apps/details?id=io.heckel.ntfy) or [iOS](https://apps.apple.com/us/app/ntfy/id1625396117) and subscribe to the topic: `http://<server-ip>:${NTFY_PORT}/${NTFY_TOPIC}`.
    - **Web Client**: Open your browser and go to `http://<server-ip>:${NTFY_PORT}/${NTFY_TOPIC}`.

By default, the `NTFY_PORT` is `8081` and the `NTFY_TOPIC` is `tw2dynalist`. You can change these in your `.env` file.

## Bookmark Management

The application provides two options for managing your Twitter bookmarks:

### 1. **Remove New Bookmarks** (`REMOVE_BOOKMARKS=true`)
- Automatically removes bookmarks from Twitter after successfully saving them to Dynalist
- Only affects newly processed tweets
- Keeps your bookmarks clean going forward

### 2. **One-time Cleanup** (`CLEANUP_PROCESSED_BOOKMARKS=true`)
- Removes ALL bookmarks that have been previously processed (exist in cache)
- Useful for cleaning up bookmarks from previous runs where removal wasn't enabled
- Runs once on startup, then continues with normal processing
- Combines well with `REMOVE_BOOKMARKS=true` for complete bookmark management

**Example usage:**
```bash
# Clean up old bookmarks and enable removal for new ones
CLEANUP_PROCESSED_BOOKMARKS=true REMOVE_BOOKMARKS=true [other vars...] go run .
```

**Safety features:**
- Bookmark removal failures don't stop the sync process
- Already processed tweets are preserved in cache even if bookmark removal fails
- 500ms delay between removals to respect API rate limits

## Automated Deployment with Portainer

This repository includes GitHub Actions for automated building and deployment:

### CI/CD Pipeline Features:
- **Multi-architecture builds**: Supports both `linux/amd64` and `linux/arm64`
- **Automatic tagging**: Uses commit SHA and latest tags
- **Deploy branch**: Updates `docker-compose.yml` with specific image tags
- **Portainer integration**: Triggers webhook for automatic redeployment

### Setup for Automated Deployment:

1. **Configure GitHub Secrets** in your repository settings:
   ```
   PORTAINER_WEBHOOK_URL=https://your-portainer-instance.com/api/webhooks/your-webhook-id
   PORTAINER_WEBHOOK_SECRET=your-webhook-secret (optional)
   ```

2. **Create Portainer Stack**:
   - Use the `deploy` branch for your Portainer stack
   - Set Git repository to: `https://github.com/your-username/tw2dynalist.git`
   - Set branch to: `deploy`
   - Enable automatic updates via webhook

3. **Workflow Triggers**:
   - **Push to master/main**: Builds, pushes image, updates deploy branch, triggers Portainer
   - **Pull Requests**: Builds and tests without deployment

### Manual Deployment:
```bash
# Deploy latest from deploy branch
git checkout deploy
docker-compose pull && docker-compose up -d
```

## Running with Docker (Manual)

```bash
docker run -d \
  --name tw2dynalist \
  -e DYNALIST_TOKEN=your_dynalist_token \
  -e TWITTER_CLIENT_ID=your_twitter_client_id \
  -e TWITTER_CLIENT_SECRET=your_twitter_client_secret \
  -e TWITTER_REDIRECT_URL=http://localhost:8080/callback \
  -e TW_USER=your_twitter_username \
  -e LOG_LEVEL=INFO \
  -e TOKEN_FILE_PATH=/app/data/token.json \
  -e CACHE_FILE_PATH=/app/data/cache.json \
  -p 8080:8080 \
  -v ./data:/app/data \
  ghcr.io/korjavin/tw2dynalist:latest
```

## Building from Source

```bash
# Clone the repository
git clone https://github.com/korjavin/tw2dynalist.git
cd tw2dynalist

# Build the binary
go build -o tw2dynalist

# Run the bot
export DYNALIST_TOKEN=your_dynalist_token
export TWITTER_CLIENT_ID=your_twitter_client_id
export TWITTER_CLIENT_SECRET=your_twitter_client_secret
export TWITTER_REDIRECT_URL=your_twitter_redirect_url  
export TW_USER=your_twitter_username
export TOKEN_FILE_PATH=token.json
./tw2dynalist
```

## Running Tests

To run the unit tests for all packages, use the following command:

```bash
go test ./...
```

## Docker Image

The Docker image is automatically built and published to GitHub Container Registry (ghcr.io) on every commit to the master branch.

You can pull the latest image with:

```bash
docker pull ghcr.io/korjavin/tw2dynalist:latest
```

## Troubleshooting

### Logging

The bot supports different logging levels to help with troubleshooting:

- `DEBUG`: Verbose logging of all operations
- `INFO`: Standard operational logging (default)
- `WARN`: Only warnings and errors
- `ERROR`: Only errors

Set the `LOG_LEVEL` environment variable to change the logging level:

```bash
export LOG_LEVEL=DEBUG
```
### Common Issues

1. **Authentication Errors**:
   - Make sure your Twitter API credentials are correct and have the necessary permissions.
   - The app uses OAuth 2.0 authentication with Twitter API v2, which requires user context authentication.
   - Ensure you've completed the OAuth 2.0 authorization flow by visiting the URL provided when running the app.
   - If you're getting a 401 Unauthorized error, check that your tokens are valid and have not expired.
   - If you're getting a 403 Forbidden error with "Unsupported Authentication", make sure you've selected the correct scopes during OAuth 2.0 setup.

2. **Rate Limiting**: Twitter API has rate limits. If you're experiencing issues, try increasing the check interval.

3. **No Bookmarks Found**: Ensure the Twitter username is correct and that the account has bookmarked tweets.

4. **API Version Issues**: This app uses Twitter API v2 endpoints. If you encounter any issues related to API endpoints, ensure your Twitter Developer account has access to the v2 API.

4. **OAuth 2.0 Flow Issues**:
   - If you're having trouble with the OAuth 2.0 flow, make sure your callback URL is correctly set in the Twitter Developer Portal and matches your `TWITTER_REDIRECT_URL` environment variable.
   - The app automatically handles the callback - you don't need to manually enter codes anymore.
   - If you need to re-authorize, simply delete the `token.json` file and run the app again.
   - Ensure the callback server port (default: 8080) is not blocked by firewalls.

5. **API Version Issues**: This app uses Twitter API v2 endpoints. If you encounter any issues related to API endpoints, ensure your Twitter Developer account has access to the v2 API.

## License

MIT