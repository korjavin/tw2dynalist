package config

import (
	"fmt"
	"os"
	"time"
)

// Config holds all configuration for the application.
type Config struct {
	DynalistToken             string
	TwitterClientID           string
	TwitterClientSecret       string
	TwitterRedirectURL        string
	TwitterUsername           string
	CacheFilePath             string
	TokenFilePath             string
	CheckInterval             time.Duration
	LogLevel                  string
	RemoveBookmarks           bool
	CleanupProcessedBookmarks bool
	CallbackPort              string
	NtfyServer                string
	NtfyTopic                 string
}

// Load reads configuration from environment variables and returns a Config struct.
func Load() (*Config, error) {
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

	cleanupProcessedBookmarksStr := os.Getenv("CLEANUP_PROCESSED_BOOKMARKS")
	cleanupProcessedBookmarks := cleanupProcessedBookmarksStr == "true"

	callbackPort := os.Getenv("CALLBACK_PORT")
	if callbackPort == "" {
		callbackPort = "8080"
	}

	ntfyServer := os.Getenv("NTFY_SERVER")
	if ntfyServer == "" {
		ntfyServer = "http://ntfy:80"
	}
	ntfyTopic := os.Getenv("NTFY_TOPIC")
	if ntfyTopic == "" {
		ntfyTopic = "tw2dynalist"
	}

	return &Config{
		DynalistToken:             dynalistToken,
		TwitterClientID:           twitterClientID,
		TwitterClientSecret:       twitterClientSecret,
		TwitterRedirectURL:        twitterRedirectURL,
		TwitterUsername:           twitterUsername,
		CacheFilePath:             cacheFilePath,
		TokenFilePath:             tokenFilePath,
		CheckInterval:             checkInterval,
		LogLevel:                  logLevel,
		RemoveBookmarks:           removeBookmarks,
		CleanupProcessedBookmarks: cleanupProcessedBookmarks,
		CallbackPort:              callbackPort,
		NtfyServer:                ntfyServer,
		NtfyTopic:                 ntfyTopic,
	}, nil
}
