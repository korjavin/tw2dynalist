package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Set up environment variables for testing
	os.Setenv("DYNALIST_TOKEN", "test_dynalist_token")
	os.Setenv("TWITTER_CLIENT_ID", "test_twitter_client_id")
	os.Setenv("TWITTER_CLIENT_SECRET", "test_twitter_client_secret")
	os.Setenv("TWITTER_REDIRECT_URL", "http://localhost:8080/callback")
	os.Setenv("TW_USER", "test_user")
	os.Setenv("CACHE_FILE_PATH", "/tmp/cache.json")
	os.Setenv("TOKEN_FILE_PATH", "/tmp/token.json")
	os.Setenv("CHECK_INTERVAL", "5m")
	os.Setenv("LOG_LEVEL", "DEBUG")
	os.Setenv("REMOVE_BOOKMARKS", "true")
	os.Setenv("CLEANUP_PROCESSED_BOOKMARKS", "true")
	os.Setenv("CALLBACK_PORT", "8888")

	// Unset environment variables after the test
	defer func() {
		os.Unsetenv("DYNALIST_TOKEN")
		os.Unsetenv("TWITTER_CLIENT_ID")
		os.Unsetenv("TWITTER_CLIENT_SECRET")
		os.Unsetenv("TWITTER_REDIRECT_URL")
		os.Unsetenv("TW_USER")
		os.Unsetenv("CACHE_FILE_PATH")
		os.Unsetenv("TOKEN_FILE_PATH")
		os.Unsetenv("CHECK_INTERVAL")
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("REMOVE_BOOKMARKS")
		os.Unsetenv("CLEANUP_PROCESSED_BOOKMARKS")
		os.Unsetenv("CALLBACK_PORT")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned an error: %v", err)
	}

	if cfg.DynalistToken != "test_dynalist_token" {
		t.Errorf("expected DynalistToken to be 'test_dynalist_token', got '%s'", cfg.DynalistToken)
	}
	if cfg.TwitterClientID != "test_twitter_client_id" {
		t.Errorf("expected TwitterClientID to be 'test_twitter_client_id', got '%s'", cfg.TwitterClientID)
	}
	if cfg.TwitterClientSecret != "test_twitter_client_secret" {
		t.Errorf("expected TwitterClientSecret to be 'test_twitter_client_secret', got '%s'", cfg.TwitterClientSecret)
	}
	if cfg.TwitterRedirectURL != "http://localhost:8080/callback" {
		t.Errorf("expected TwitterRedirectURL to be 'http://localhost:8080/callback', got '%s'", cfg.TwitterRedirectURL)
	}
	if cfg.TwitterUsername != "test_user" {
		t.Errorf("expected TwitterUsername to be 'test_user', got '%s'", cfg.TwitterUsername)
	}
	if cfg.CacheFilePath != "/tmp/cache.json" {
		t.Errorf("expected CacheFilePath to be '/tmp/cache.json', got '%s'", cfg.CacheFilePath)
	}
	if cfg.TokenFilePath != "/tmp/token.json" {
		t.Errorf("expected TokenFilePath to be '/tmp/token.json', got '%s'", cfg.TokenFilePath)
	}
	if cfg.CheckInterval != 5*time.Minute {
		t.Errorf("expected CheckInterval to be '5m', got '%s'", cfg.CheckInterval)
	}
	if cfg.LogLevel != "DEBUG" {
		t.Errorf("expected LogLevel to be 'DEBUG', got '%s'", cfg.LogLevel)
	}
	if !cfg.RemoveBookmarks {
		t.Errorf("expected RemoveBookmarks to be true, got false")
	}
	if !cfg.CleanupProcessedBookmarks {
		t.Errorf("expected CleanupProcessedBookmarks to be true, got false")
	}
	if cfg.CallbackPort != "8888" {
		t.Errorf("expected CallbackPort to be '8888', got '%s'", cfg.CallbackPort)
	}
}
