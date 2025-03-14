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

// OAuth2Token represents an OAuth2 token
type OAuth2Token struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	RefreshToken string    `json:"refresh_token"`
	Expiry       time.Time `json:"expiry"`
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

// SaveToken saves the OAuth2 token to a file
func SaveToken(filePath string, token *oauth2.Token) error {
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %v", err)
	}

	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %v", err)
	}

	return nil
}

// LoadToken loads the OAuth2 token from a file
func LoadToken(filePath string) (*oauth2.Token, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read token file: %v", err)
	}

	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to parse token file: %v", err)
	}

	return &token, nil
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

	if result["_code"] != "Ok" {
		return fmt.Errorf("dynalist API error: %v", result)
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
	var err error

	if _, err := os.Stat(config.TokenFilePath); os.IsNotExist(err) {
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

		logger.Info("Please visit the following URL to authorize this application:")
		authURL := GetAuthURL(oauth2Config, "state", codeChallenge)
		logger.Info("%s", authURL)

		// Wait for the user to enter the authorization code
		logger.Info("Enter the authorization code: ")
		var code string
		fmt.Scanln(&code)

		// Exchange the authorization code for a token using the code verifier
		token, err = ExchangeToken(oauth2Config, code, codeVerifier)
		if err != nil {
			return nil, fmt.Errorf("failed to exchange token: %v", err)
		}

		// Save the token
		if err := SaveToken(config.TokenFilePath, token); err != nil {
			logger.Warn("Failed to save token: %v", err)
		}
	} else {
		// Load the token from the file
		token, err = LoadToken(config.TokenFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load token: %v", err)
		}
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

	// Get user ID from username
	logger.Info("Looking up user ID for username: %s", config.TwitterUsername)
	opts := twitterv2.UserLookupOpts{
		UserFields: []twitterv2.UserField{twitterv2.UserFieldID, twitterv2.UserFieldName, twitterv2.UserFieldUserName},
	}

	ctx := context.Background()
	userResponse, err := client.UserNameLookup(ctx, []string{config.TwitterUsername}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup Twitter user: %v", err)
	}

	if len(userResponse.Raw.Users) == 0 {
		return nil, fmt.Errorf("user not found: %s", config.TwitterUsername)
	}

	userID := userResponse.Raw.Users[0].ID
	logger.Info("Authenticated as Twitter user: @%s (ID: %s)", config.TwitterUsername, userID)

	return &TwitterClient{
		client: client,
		userID: userID,
	}, nil
}

// GetBookmarks retrieves bookmarked tweets for a user
func (t *TwitterClient) GetBookmarks(username string, logger *Logger) ([]Tweet, error) {
	logger.Info("Fetching bookmarks for user: %s", username)

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
	bookmarksResponse, err := t.client.TweetBookmarksLookup(ctx, t.userID, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get bookmarks: %v", err)
	}

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
	tweets, err := twitterClient.GetBookmarks(config.TwitterUsername, logger)
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

		if err := dynalistClient.AddToInbox(content, note, logger); err != nil {
			logger.Error("Error adding tweet %s to Dynalist: %v", tweet.ID, err)
			failed++
			continue
		}

		// Mark as processed
		cache.MarkProcessed(tweet.ID)
		logger.Info("Successfully added tweet %s to Dynalist", tweet.ID)
		processed++
	}

	logger.Info("Bookmark processing complete. Processed: %d, Skipped: %d, Failed: %d", processed, skipped, failed)
	return nil
}
