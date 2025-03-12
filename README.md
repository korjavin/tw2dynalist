# Twitter to Dynalist Bot

This bot monitors a Twitter user's bookmarks and automatically adds them to your Dynalist inbox.

## Features

- Monitors a specified Twitter user's bookmarks
- Adds new bookmarked tweets to Dynalist inbox
- Uses local cache to avoid duplicates
- Checks for new bookmarks hourly (configurable)
- Runs in a Docker container

## Prerequisites

- Twitter API credentials (API key, API secret, Access token, Access secret)
- Dynalist API token
- Docker (for containerized deployment)

## Environment Variables

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `DYNALIST_TOKEN` | Your Dynalist API token | Yes | - |
| `TWITTER_API_KEY` | Twitter API key | Yes | - |
| `TWITTER_API_SECRET` | Twitter API secret | Yes | - |
| `TWITTER_ACCESS_TOKEN` | Twitter Access token | Yes | - |
| `TWITTER_ACCESS_SECRET` | Twitter Access secret | Yes | - |
| `TW_USER` | Twitter username to monitor | Yes | - |
| `CACHE_FILE_PATH` | Path to cache file | No | `cache.json` |
| `CHECK_INTERVAL` | Interval to check for new bookmarks | No | `1h` |
| `LOG_LEVEL` | Logging level (DEBUG, INFO, WARN, ERROR) | No | `INFO` |

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
   - Set the callback URL (e.g., http://localhost:8080/callback)
   - Select the required scopes: tweet.read, users.read, bookmark.read
   - Save your changes

5. Get your OAuth 2.0 credentials:
   - In your app settings, navigate to the "Keys and tokens" tab
   - You'll find your Client ID and Client Secret under "OAuth 2.0 Client ID and Client Secret"
   - Make sure to save these values as they will be needed to configure the bot

6. Additional requirements:
   - Ensure your Twitter account has the necessary permissions to access the API
   - For production use, you may need to apply for Elevated access to the Twitter API
   - The app uses Twitter API v2 endpoints with OAuth 2.0 User Context authentication

## Getting Dynalist API Token

1. Log in to your Dynalist account
2. Go to Settings > Developer
3. Copy your API token

## Running with Docker

```bash
docker run -d \
  --name tw2dynalist \
  -e DYNALIST_TOKEN=your_dynalist_token \
  -e TWITTER_CLIENT_ID=your_twitter_client_id \
  -e TWITTER_CLIENT_SECRET=your_twitter_client_secret \
  -e TWITTER_REDIRECT_URL=your_twitter_redirect_url \
  -e TW_USER=your_twitter_username \
  -e LOG_LEVEL=INFO \
  -e TOKEN_FILE_PATH=/app/token.json \
  -v /path/to/cache:/app/cache \
  -v /path/to/token:/app/token.json \
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

5. **OAuth 2.0 Flow Issues**:
   - If you're having trouble with the OAuth 2.0 flow, make sure your callback URL is correctly set in the Twitter Developer Portal.
   - The authorization code must be entered exactly as provided in the callback URL.
   - If you need to re-authorize, simply delete the token.json file and run the app again.
4. **API Version Issues**: This app now uses Twitter API v2 endpoints. If you encounter any issues related to API endpoints, ensure your Twitter Developer account has access to the v2 API.

## License

MIT