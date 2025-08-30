package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	twitterv2 "github.com/g8rswimmer/go-twitter/v2"
	"golang.org/x/oauth2"
)

// Configuration holds all environment variables
type Configuration struct {
	DynalistToken       string
	TwitterClientID     string
	TwitterClientSecret string
	TwitterRedirectURL  string
	TwitterUsername     string
	CacheFilePath       string
	CheckInterval       time.Duration
	LogLevel            string
	TokenFilePath       string // Path to store the OAuth2 token
	RemoveBookmarks     bool   // Whether to remove bookmarks after successful Dynalist save
}

// Cache represents the structure to store processed tweets
type Cache struct {
	ProcessedTweets map[string]bool `json:"processed_tweets"`
	mu              sync.Mutex
}

// DynalistClient handles interactions with Dynalist API
type DynalistClient struct {
	token  string
	client *http.Client
}

// DynalistInboxRequest represents the request to add an item to Dynalist inbox
type DynalistInboxRequest struct {
	Token   string `json:"token"`
	Content string `json:"content"`
	Note    string `json:"note,omitempty"`
}

// TwitterClient wraps the Twitter API client
type TwitterClient struct {
	client *twitterv2.Client
	userID string
}

// OAuth2Token represents an OAuth2 token with user info
type OAuth2Token struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	RefreshToken string    `json:"refresh_token"`
	Expiry       time.Time `json:"expiry"`
	UserID       string    `json:"user_id,omitempty"`
}

// OAuth2Authorizer implements the Authorizer interface for OAuth2
type OAuth2Authorizer struct {
	token *oauth2.Token
}

// Add adds the OAuth2 authorization to the request
func (a *OAuth2Authorizer) Add(req *http.Request) {
	// Add the Authorization header with the access token
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.token.AccessToken))
}

// TokenSource is a source of OAuth2 tokens
type TokenSource struct {
	token *oauth2.Token
}

// Token returns the current token
func (t *TokenSource) Token() (*oauth2.Token, error) {
	return t.token, nil
}

// SaveTokenWithUserInfo saves the OAuth2 token with user ID to a file
func SaveTokenWithUserInfo(filePath string, token *oauth2.Token, userID string) error {
	tokenData := OAuth2Token{
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
		UserID:       userID,
	}

	data, err := json.MarshalIndent(tokenData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %v", err)
	}

	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %v", err)
	}

	return nil
}

// SaveToken saves the OAuth2 token to a file (without user info)
func SaveToken(filePath string, token *oauth2.Token) error {
	return SaveTokenWithUserInfo(filePath, token, "")
}

// LoadTokenWithUserInfo loads the OAuth2 token and user ID from a file
func LoadTokenWithUserInfo(filePath string) (*oauth2.Token, string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read token file: %v", err)
	}

	// Parse the token data
	var tokenData OAuth2Token
	if err := json.Unmarshal(data, &tokenData); err != nil {
		return nil, "", fmt.Errorf("failed to parse token file: %v", err)
	}

	token := &oauth2.Token{
		AccessToken:  tokenData.AccessToken,
		TokenType:    tokenData.TokenType,
		RefreshToken: tokenData.RefreshToken,
		Expiry:       tokenData.Expiry,
	}

	return token, tokenData.UserID, nil
}

// LoadToken loads the OAuth2 token from a file (backward compatibility)
func LoadToken(filePath string) (*oauth2.Token, error) {
	token, _, err := LoadTokenWithUserInfo(filePath)
	return token, err
}

// GenerateCodeVerifier creates a code verifier for PKCE
func GenerateCodeVerifier() (string, error) {
	// Generate a random string of 32-64 characters
	b := make([]byte, 32) // 32 bytes = 256 bits
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %v", err)
	}

	// Base64 URL encode the random bytes
	verifier := base64.RawURLEncoding.EncodeToString(b)

	// The code verifier should only contain alphanumeric characters, hyphens, underscores, periods, and tildes
	// But base64 URL encoding already ensures this, so no additional cleaning is needed

	return verifier, nil
}

// GenerateCodeChallenge creates a code challenge from a code verifier
func GenerateCodeChallenge(verifier string) string {
	// Create SHA256 hash of the verifier
	hash := sha256.Sum256([]byte(verifier))

	// Base64 URL encode the hash
	challenge := base64.RawURLEncoding.EncodeToString(hash[:])

	return challenge
}

// GetAuthURL returns the URL to redirect the user to for OAuth2 authentication with PKCE
func GetAuthURL(config *oauth2.Config, state string, codeChallenge string) string {
	// Add PKCE parameters to the auth URL
	opts := []oauth2.AuthCodeOption{
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	}

	authURL := config.AuthCodeURL(state, opts...)

	// Log the raw URL for debugging
	fmt.Printf("DEBUG: Raw auth URL with PKCE: %s\n", authURL)

	return authURL
}

// ExchangeToken exchanges an authorization code for an OAuth2 token with PKCE
func ExchangeToken(config *oauth2.Config, code string, codeVerifier string) (*oauth2.Token, error) {
	ctx := context.Background()

	// Add the code verifier to the token request
	opts := []oauth2.AuthCodeOption{
		oauth2.SetAuthURLParam("code_verifier", codeVerifier),
	}

	return config.Exchange(ctx, code, opts...)
}

// StartCallbackServer starts an HTTP server to handle OAuth callbacks
func StartCallbackServer(port string, codeChan chan string, stateChan chan string, errorChan chan error) *http.Server {
	mux := http.NewServeMux()
	
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received callback request: %s", r.URL.String())
		
		// Parse query parameters
		query := r.URL.Query()
		code := query.Get("code")
		state := query.Get("state")
		errorParam := query.Get("error")
		
		log.Printf("Callback parameters - code: %s, state: %s, error: %s", 
			code[:min(len(code), 10)]+"...", state, errorParam)
		
		if errorParam != "" {
			errorDescription := query.Get("error_description")
			errorMsg := fmt.Sprintf("OAuth Error: %s - %s", errorParam, errorDescription)
			log.Printf("OAuth callback error: %s", errorMsg)
			http.Error(w, errorMsg, http.StatusBadRequest)
			errorChan <- fmt.Errorf("OAuth error: %s - %s", errorParam, errorDescription)
			return
		}
		
		if code == "" {
			log.Printf("Authorization code not found in callback")
			http.Error(w, "Authorization code not found", http.StatusBadRequest)
			errorChan <- fmt.Errorf("authorization code not found in callback")
			return
		}
		
		// Send success response
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
			<html>
			<head><title>Authorization Successful</title></head>
			<body>
				<h1>✅ Authorization Successful!</h1>
				<p>You can close this window and return to your application.</p>
				<p>The authorization code has been received and your app is now authenticating...</p>
				<script>setTimeout(function(){ window.close(); }, 5000);</script>
			</body>
			</html>
		`))
		
		log.Printf("Sending authorization code to application...")
		// Send the code and state to the channels
		codeChan <- code
		stateChan <- state
	})
	
	// Add a health check endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
			<html>
			<head><title>OAuth Callback Server</title></head>
			<body>
				<h1>OAuth Callback Server Running</h1>
				<p>This server is waiting for OAuth callbacks on /callback</p>
			</body>
			</html>
		`))
	})
	
	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}
	
	go func() {
		log.Printf("Starting callback server on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Callback server error: %v", err)
			errorChan <- fmt.Errorf("failed to start callback server: %v", err)
		}
	}()
	
	return server
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ExtractPortFromURL extracts the port from a URL, returns "8080" as default
func ExtractPortFromURL(redirectURL string) (string, error) {
	parsedURL, err := url.Parse(redirectURL)
	if err != nil {
		return "", fmt.Errorf("invalid redirect URL: %v", err)
	}
	
	port := parsedURL.Port()
	if port == "" {
		// Default ports based on scheme
		if parsedURL.Scheme == "https" {
			port = "443"
		} else {
			port = "8080"
		}
	}
	
	return port, nil
}

// Tweet represents a simplified tweet structure
type Tweet struct {
	ID   string
	Text string
	URL  string
}

// Logger provides different log levels
type Logger struct {
	level string
}

// NewLogger creates a new logger with the specified level
func NewLogger(level string) *Logger {
	return &Logger{
		level: strings.ToUpper(level),
	}
}

// Debug logs debug messages
func (l *Logger) Debug(format string, v ...interface{}) {
	if l.level == "DEBUG" {
		log.Printf("[DEBUG] "+format, v...)
	}
}

// Info logs info messages
func (l *Logger) Info(format string, v ...interface{}) {
	if l.level == "DEBUG" || l.level == "INFO" {
		log.Printf("[INFO] "+format, v...)
	}
}

// Warn logs warning messages
func (l *Logger) Warn(format string, v ...interface{}) {
	if l.level == "DEBUG" || l.level == "INFO" || l.level == "WARN" {
		log.Printf("[WARN] "+format, v...)
	}
}

// Error logs error messages
func (l *Logger) Error(format string, v ...interface{}) {
	log.Printf("[ERROR] "+format, v...)
}

// Fatal logs fatal messages and exits
func (l *Logger) Fatal(format string, v ...interface{}) {
	log.Fatalf("[FATAL] "+format, v...)
}

// NewConfiguration loads configuration from environment variables
func NewConfiguration() (*Configuration, error) {
	dynalistToken := os.Getenv("DYNALIST_TOKEN")
	if dynalistToken == "" {
		return nil, fmt.Errorf("DYNALIST_TOKEN environment variable is required")
	}

	twitterClientID := os.Getenv("TWITTER_CLIENT_ID")
	if twitterClientID == "" {
		return nil, fmt.Errorf("TWITTER_CLIENT_ID environment variable is required")
	}

	twitterClientSecret := os.Getenv("TWITTER_CLIENT_SECRET")
	if twitterClientSecret == "" {
		return nil, fmt.Errorf("TWITTER_CLIENT_SECRET environment variable is required")
	}

	twitterRedirectURL := os.Getenv("TWITTER_REDIRECT_URL")
	if twitterRedirectURL == "" {
		return nil, fmt.Errorf("TWITTER_REDIRECT_URL environment variable is required")
	}

	twitterUsername := os.Getenv("TW_USER")
	if twitterUsername == "" {
		return nil, fmt.Errorf("TW_USER environment variable is required")
	}

	cacheFilePath := os.Getenv("CACHE_FILE_PATH")
	if cacheFilePath == "" {
		cacheFilePath = "cache.json"
	}

	tokenFilePath := os.Getenv("TOKEN_FILE_PATH")
	if tokenFilePath == "" {
		tokenFilePath = "token.json"
	}

	checkIntervalStr := os.Getenv("CHECK_INTERVAL")
	var checkInterval time.Duration
	if checkIntervalStr == "" {
		checkInterval = 1 * time.Hour
	} else {
		var err error
		checkInterval, err = time.ParseDuration(checkIntervalStr)
		if err != nil {
			return nil, fmt.Errorf("invalid CHECK_INTERVAL format: %v", err)
		}
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "INFO"
	}

	removeBookmarksStr := os.Getenv("REMOVE_BOOKMARKS")
	removeBookmarks := removeBookmarksStr == "true"

	return &Configuration{
		DynalistToken:       dynalistToken,
		TwitterClientID:     twitterClientID,
		TwitterClientSecret: twitterClientSecret,
		TwitterRedirectURL:  twitterRedirectURL,
		TwitterUsername:     twitterUsername,
		CacheFilePath:       cacheFilePath,
		TokenFilePath:       tokenFilePath,
		CheckInterval:       checkInterval,
		LogLevel:            logLevel,
		RemoveBookmarks:     removeBookmarks,
	}, nil
}

// NewCache initializes a new cache or loads an existing one
func NewCache(filePath string, logger *Logger) (*Cache, error) {
	cache := &Cache{
		ProcessedTweets: make(map[string]bool),
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if dir != "." {
		logger.Debug("Creating cache directory: %s", dir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create cache directory: %v", err)
		}
	}

	// Try to load existing cache
	logger.Debug("Attempting to load cache from: %s", filePath)
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Cache file doesn't exist, return empty cache
			logger.Info("Cache file doesn't exist, creating new cache")
			return cache, nil
		}
		return nil, fmt.Errorf("failed to read cache file: %v", err)
	}

	// Parse cache data
	logger.Debug("Parsing cache data")
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("failed to parse cache file: %v", err)
	}

	logger.Info("Cache loaded successfully with %d processed tweets", len(cache.ProcessedTweets))
	return cache, nil
}

// SaveCache persists the cache to disk
func (c *Cache) SaveCache(filePath string, logger *Logger) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	logger.Debug("Marshaling cache data")
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %v", err)
	}

	logger.Debug("Writing cache to file: %s", filePath)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %v", err)
	}

	logger.Info("Cache saved successfully with %d processed tweets", len(c.ProcessedTweets))
	return nil
}

// MarkProcessed marks a tweet as processed
func (c *Cache) MarkProcessed(tweetID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ProcessedTweets[tweetID] = true
}

// IsProcessed checks if a tweet has been processed
func (c *Cache) IsProcessed(tweetID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ProcessedTweets[tweetID]
}

// NewDynalistClient creates a new Dynalist API client
func NewDynalistClient(token string, logger *Logger) *DynalistClient {
	logger.Debug("Creating new Dynalist client")
	return &DynalistClient{
		token:  token,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// AddToInbox adds an item to Dynalist inbox
func (d *DynalistClient) AddToInbox(content, note string, logger *Logger) error {
	logger.Debug("Preparing request to add item to Dynalist inbox")
	reqBody := DynalistInboxRequest{
		Token:   d.token,
		Content: content,
		Note:    note,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	logger.Debug("Sending request to Dynalist API")
	req, err := http.NewRequest("POST", "https://dynalist.io/api/v1/inbox/add", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}

	logger.Debug("Dynalist API response: %v", result)

	if result["_code"] != "Ok" {
		code := result["_code"]
		msg := result["_msg"]
		
		// Handle specific error codes
		switch code {
		case "TooManyRequests":
			logger.Warn("Dynalist rate limit hit, pausing for 2 seconds")
			time.Sleep(2 * time.Second)
			return fmt.Errorf("dynalist rate limit: %s", msg)
		case "InvalidToken":
			logger.Error("Dynalist token is invalid: %s", msg)
			return fmt.Errorf("dynalist invalid token: %s", msg)
		case "Unauthorized":
			logger.Error("Dynalist unauthorized: %s", msg)
			return fmt.Errorf("dynalist unauthorized: %s", msg)
		default:
			logger.Error("Dynalist API error [%s]: %s", code, msg)
			return fmt.Errorf("dynalist API error [%s]: %s", code, msg)
		}
	}

	logger.Debug("Successfully added item to Dynalist inbox")
	return nil
}

// NewTwitterClient creates a new Twitter API client
func NewTwitterClient(config *Configuration, logger *Logger) (*TwitterClient, error) {
	logger.Debug("Creating OAuth2 configuration")

	// Create OAuth2 configuration
	oauth2Config := &oauth2.Config{
		ClientID:     config.TwitterClientID,
		ClientSecret: config.TwitterClientSecret,
		RedirectURL:  config.TwitterRedirectURL,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://twitter.com/i/oauth2/authorize",
			TokenURL: "https://api.twitter.com/2/oauth2/token",
		},
		Scopes: []string{"tweet.read", "users.read", "bookmark.read"},
	}

	// Log the redirect URL for debugging
	logger.Debug("OAuth2 redirect URL: %s", config.TwitterRedirectURL)

	// Check if we have a token file
	var token *oauth2.Token
	var userID string
	var err error

	if _, statErr := os.Stat(config.TokenFilePath); os.IsNotExist(statErr) {
		// No token file, need to get a new token
		logger.Info("No token file found at %s", config.TokenFilePath)

		// Generate code verifier and challenge for PKCE
		logger.Debug("Generating PKCE code verifier and challenge")
		codeVerifier, err := GenerateCodeVerifier()
		if err != nil {
			return nil, fmt.Errorf("failed to generate code verifier: %v", err)
		}
		codeChallenge := GenerateCodeChallenge(codeVerifier)
		logger.Debug("Code verifier: %s", codeVerifier)
		logger.Debug("Code challenge: %s", codeChallenge)

		// Extract port from redirect URL for callback server
		port, err := ExtractPortFromURL(config.TwitterRedirectURL)
		if err != nil {
			return nil, fmt.Errorf("failed to extract port from redirect URL: %v", err)
		}

		// Start the callback server
		logger.Debug("Starting callback server on port %s", port)
		codeChan := make(chan string, 1)
		stateChan := make(chan string, 1)
		errorChan := make(chan error, 1)
		
		callbackServer := StartCallbackServer(port, codeChan, stateChan, errorChan)
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			callbackServer.Shutdown(ctx)
		}()

		// Wait a moment for server to start
		time.Sleep(500 * time.Millisecond)

		logger.Info("Callback server started on http://localhost:%s/callback", port)
		logger.Info("Please visit the following URL to authorize this application:")
		authURL := GetAuthURL(oauth2Config, "state", codeChallenge)
		logger.Info("%s", authURL)
		logger.Info("Waiting for OAuth callback... (this will wait indefinitely until you complete authorization)")

		// Wait for the callback with longer timeout
		var code string
		select {
		case code = <-codeChan:
			logger.Debug("Received authorization code from callback")
		case err := <-errorChan:
			return nil, fmt.Errorf("OAuth callback error: %v", err)
		case <-time.After(30 * time.Minute):
			return nil, fmt.Errorf("OAuth authorization timeout after 30 minutes")
		}

		// Exchange the authorization code for a token using the code verifier
		token, err = ExchangeToken(oauth2Config, code, codeVerifier)
		if err != nil {
			return nil, fmt.Errorf("failed to exchange token: %v", err)
		}

		// We'll need to get user info after creating the client, so save token without user info for now
		if err := SaveToken(config.TokenFilePath, token); err != nil {
			logger.Warn("Failed to save token: %v", err)
		}
	} else {
		// Load the token from the file
		token, userID, err = LoadTokenWithUserInfo(config.TokenFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load token: %v", err)
		}
		logger.Debug("Loaded token with userID: %s", userID)
	}

	// Create OAuth2 authorizer
	authorizer := &OAuth2Authorizer{
		token: token,
	}

	logger.Debug("Creating Twitter v2 client")
	client := &twitterv2.Client{
		Authorizer: authorizer,
		Client:     &http.Client{Timeout: 10 * time.Second},
		Host:       "https://api.twitter.com",
	}

	// We need the actual user ID for the bookmarks API, so let's get it
	if userID == "" || userID == "me" {
		logger.Info("Need to lookup actual user ID for bookmarks API")
		
		// Create a simple request to get the current user's information
		// We'll use a minimal request to avoid rate limits
		userOpts := twitterv2.UserLookupOpts{
			UserFields: []twitterv2.UserField{twitterv2.UserFieldID},
		}
		
		// Try to get the authenticated user's information
		// Since UserLookupMe doesn't exist, we'll need to use the authenticated user's username
		cleanUsername := strings.TrimPrefix(config.TwitterUsername, "@")
		logger.Debug("Looking up user ID for username: %s", cleanUsername)
		
		ctx := context.Background()
		userResponse, err := client.UserNameLookup(ctx, []string{cleanUsername}, userOpts)
		if err != nil {
			logger.Warn("Failed to lookup user ID, using fallback: %v", err)
			// As a last resort, we'll try without a user ID or return an error
			return nil, fmt.Errorf("cannot get user ID for bookmarks API: %v", err)
		}
		
		if userResponse.Raw == nil || len(userResponse.Raw.Users) == 0 || userResponse.Raw.Users[0] == nil {
			return nil, fmt.Errorf("failed to get user information for bookmarks API")
		}
		
		userID = userResponse.Raw.Users[0].ID
		logger.Info("Successfully retrieved user ID: %s", userID)
		
		// Save the token with actual user ID
		if err := SaveTokenWithUserInfo(config.TokenFilePath, token, userID); err != nil {
			logger.Warn("Failed to save token with user info: %v", err)
		}
	} else {
		logger.Info("Using cached user ID: %s", userID)
	}

	return &TwitterClient{
		client: client,
		userID: userID,
	}, nil
}

// GetBookmarks retrieves bookmarked tweets for the authenticated user
func (t *TwitterClient) GetBookmarks(logger *Logger) ([]Tweet, error) {
	logger.Info("Fetching bookmarks for user ID: %s", t.userID)

	// Use the Twitter API v2 bookmarks endpoint
	logger.Debug("Fetching user bookmarks using v2 API")
	opts := twitterv2.TweetBookmarksLookupOpts{
		MaxResults: 100, // Maximum allowed by Twitter API v2
		TweetFields: []twitterv2.TweetField{
			twitterv2.TweetFieldID,
			twitterv2.TweetFieldText,
			twitterv2.TweetFieldAuthorID,
			twitterv2.TweetFieldCreatedAt,
		},
		UserFields: []twitterv2.UserField{
			twitterv2.UserFieldID,
			twitterv2.UserFieldName,
			twitterv2.UserFieldUserName,
		},
		Expansions: []twitterv2.Expansion{
			twitterv2.ExpansionAuthorID,
		},
	}

	ctx := context.Background()
	logger.Debug("Making TweetBookmarksLookup API call with userID: %s", t.userID)
	logger.Debug("API request options: MaxResults=%d, TweetFields=%v", opts.MaxResults, opts.TweetFields)
	
	bookmarksResponse, err := t.client.TweetBookmarksLookup(ctx, t.userID, opts)
	if err != nil {
		logger.Error("TweetBookmarksLookup failed: %v", err)
		
		// Check if it's a rate limit error
		if strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "Too Many Requests") {
			logger.Warn("Twitter API rate limit hit on bookmarks endpoint. This is likely due to frequent testing.")
			logger.Info("The app will continue running and retry on the next check cycle (every %v)", time.Duration(10)*time.Minute)
			return []Tweet{}, nil // Return empty list instead of failing
		}
		
		return nil, fmt.Errorf("failed to get bookmarks: %v", err)
	}
	
	logger.Debug("API call successful, checking response...")

	if bookmarksResponse.Raw == nil || len(bookmarksResponse.Raw.Tweets) == 0 {
		logger.Info("No bookmarks found")
		return []Tweet{}, nil
	}

	logger.Info("Found %d bookmarks", len(bookmarksResponse.Raw.Tweets))

	// Create a map of author IDs to usernames
	authorMap := make(map[string]string)
	if bookmarksResponse.Raw.Includes != nil && len(bookmarksResponse.Raw.Includes.Users) > 0 {
		for _, user := range bookmarksResponse.Raw.Includes.Users {
			authorMap[user.ID] = user.UserName
		}
	}

	var tweets []Tweet
	for _, tweet := range bookmarksResponse.Raw.Tweets {
		// Get the username from the author map, or use "user" if not found
		username := "user"
		if authorUsername, ok := authorMap[tweet.AuthorID]; ok {
			username = authorUsername
		}

		tweetURL := fmt.Sprintf("https://twitter.com/%s/status/%s", username, tweet.ID)
		tweets = append(tweets, Tweet{
			ID:   tweet.ID,
			Text: tweet.Text,
			URL:  tweetURL,
		})
		logger.Debug("Processed tweet: %s", tweet.ID)
	}

	return tweets, nil
}

// RemoveBookmark removes a tweet from bookmarks using direct HTTP call
func (t *TwitterClient) RemoveBookmark(tweetID string, logger *Logger) error {
	logger.Debug("Attempting to remove bookmark for tweet ID: %s", tweetID)
	
	// Use direct HTTP call to Twitter API v2 DELETE /2/users/:id/bookmarks/:tweet_id
	url := fmt.Sprintf("https://api.twitter.com/2/users/%s/bookmarks/%s", t.userID, tweetID)
	
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create remove bookmark request: %v", err)
	}
	
	// Add authorization header using the same token as the client
	// We need to get the token from the client's authorizer
	if authorizer, ok := t.client.Authorizer.(*OAuth2Authorizer); ok {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authorizer.token.AccessToken))
	} else {
		return fmt.Errorf("unable to get authorization token for bookmark removal")
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send remove bookmark request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == 200 || resp.StatusCode == 204 {
		logger.Debug("Successfully removed bookmark for tweet %s", tweetID)
		return nil
	}
	
	// Read response body for error details
	body, _ := io.ReadAll(resp.Body)
	logger.Debug("Remove bookmark response: %s", string(body))
	
	if resp.StatusCode == 404 {
		logger.Debug("Bookmark for tweet %s was not found (already removed or never bookmarked)", tweetID)
		return nil // Consider this a success
	}
	
	return fmt.Errorf("failed to remove bookmark (HTTP %d): %s", resp.StatusCode, string(body))
}

func main() {
	// Set up logging
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Println("Starting Twitter to Dynalist bot")

	// Load configuration
	config, err := NewConfiguration()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create logger
	logger := NewLogger(config.LogLevel)
	logger.Info("Log level set to: %s", config.LogLevel)

	// Initialize cache
	logger.Info("Initializing cache from: %s", config.CacheFilePath)
	cache, err := NewCache(config.CacheFilePath, logger)
	if err != nil {
		logger.Fatal("Failed to initialize cache: %v", err)
	}

	// Initialize clients
	logger.Info("Initializing Dynalist client")
	dynalistClient := NewDynalistClient(config.DynalistToken, logger)

	logger.Info("Initializing Twitter client")
	twitterClient, err := NewTwitterClient(config, logger)
	if err != nil {
		logger.Fatal("Failed to initialize Twitter client: %v", err)
	}

	// Process bookmarks initially
	logger.Info("Processing bookmarks initially")
	if err := processBookmarks(twitterClient, dynalistClient, cache, config, logger); err != nil {
		logger.Error("Error processing bookmarks: %v", err)
	}

	// Save cache after initial processing
	logger.Info("Saving cache after initial processing")
	if err := cache.SaveCache(config.CacheFilePath, logger); err != nil {
		logger.Error("Error saving cache: %v", err)
	}

	// Set up ticker for periodic checks
	ticker := time.NewTicker(config.CheckInterval)
	defer ticker.Stop()

	logger.Info("Bot started. Checking for new bookmarks every %v", config.CheckInterval)

	// Main loop
	for {
		select {
		case <-ticker.C:
			logger.Info("Checking for new bookmarks...")
			if err := processBookmarks(twitterClient, dynalistClient, cache, config, logger); err != nil {
				logger.Error("Error processing bookmarks: %v", err)
				continue
			}

			// Save cache after processing
			logger.Info("Saving cache after processing")
			if err := cache.SaveCache(config.CacheFilePath, logger); err != nil {
				logger.Error("Error saving cache: %v", err)
			}
		}
	}
}

// processBookmarks retrieves and processes bookmarked tweets
func processBookmarks(twitterClient *TwitterClient, dynalistClient *DynalistClient, cache *Cache, config *Configuration, logger *Logger) error {
	logger.Info("Starting to process bookmarks")
	tweets, err := twitterClient.GetBookmarks(logger)
	if err != nil {
		return fmt.Errorf("failed to get bookmarks: %v", err)
	}

	logger.Info("Found %d bookmarked tweets", len(tweets))

	var processed int
	var skipped int
	var failed int

	for _, tweet := range tweets {
		if cache.IsProcessed(tweet.ID) {
			logger.Debug("Tweet %s already processed, skipping", tweet.ID)
			skipped++
			continue
		}

		logger.Info("Processing tweet: %s", tweet.ID)
		logger.Debug("Tweet text: %s", tweet.Text)
		logger.Debug("Tweet URL: %s", tweet.URL)

		// Add to Dynalist inbox
		content := fmt.Sprintf("Tweet: %s", tweet.Text)
		note := fmt.Sprintf("URL: %s", tweet.URL)

		// Retry logic for rate limiting
		maxRetries := 3
		var err error
		for retry := 0; retry < maxRetries; retry++ {
			err = dynalistClient.AddToInbox(content, note, logger)
			if err == nil {
				break
			}
			
			// Check if it's a rate limit error and retry
			if strings.Contains(err.Error(), "rate limit") && retry < maxRetries-1 {
				logger.Warn("Rate limited on tweet %s, retrying in %d seconds (attempt %d/%d)", 
					tweet.ID, (retry+1)*2, retry+1, maxRetries)
				time.Sleep(time.Duration((retry+1)*2) * time.Second)
				continue
			}
			
			// If it's not a rate limit error, don't retry
			if !strings.Contains(err.Error(), "rate limit") {
				break
			}
		}
		
		if err != nil {
			logger.Error("Error adding tweet %s to Dynalist after %d attempts: %v", tweet.ID, maxRetries, err)
			failed++
			continue
		}

		// Mark as processed
		cache.MarkProcessed(tweet.ID)
		logger.Info("Successfully added tweet %s to Dynalist", tweet.ID)
		processed++
		
		// Remove bookmark if configured to do so
		if config.RemoveBookmarks {
			if err := twitterClient.RemoveBookmark(tweet.ID, logger); err != nil {
				logger.Warn("Failed to remove bookmark for tweet %s: %v", tweet.ID, err)
				// Don't fail the whole process if bookmark removal fails
			} else {
				logger.Info("Removed bookmark for tweet %s", tweet.ID)
			}
		}
		
		// Add a small pause to avoid rate limiting
		time.Sleep(200 * time.Millisecond)
	}

	logger.Info("Bookmark processing complete. Processed: %d, Skipped: %d, Failed: %d", processed, skipped, failed)
	return nil
}
