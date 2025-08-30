package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/korjavin/tw2dynalist/internal/logger"
)

func TestFileStorage(t *testing.T) {
	log := logger.New("DEBUG")
	tempDir := t.TempDir()
	cacheFile := filepath.Join(tempDir, "cache.json")

	// Test NewFileStorage
	storage, err := NewFileStorage(cacheFile, log)
	if err != nil {
		t.Fatalf("NewFileStorage() returned an error: %v", err)
	}
	if storage == nil {
		t.Fatal("NewFileStorage() returned a nil storage")
	}

	// Test IsProcessed on an empty cache
	if storage.IsProcessed("123") {
		t.Error("IsProcessed() should return false for a new tweet")
	}

	// Test MarkProcessed
	storage.MarkProcessed("123")
	if !storage.IsProcessed("123") {
		t.Error("IsProcessed() should return true for a processed tweet")
	}

	// Test Save
	if err := storage.Save(); err != nil {
		t.Fatalf("Save() returned an error: %v", err)
	}

	// Test loading from an existing file
	newStorage, err := NewFileStorage(cacheFile, log)
	if err != nil {
		t.Fatalf("NewFileStorage() returned an error when loading: %v", err)
	}
	if !newStorage.IsProcessed("123") {
		t.Error("IsProcessed() should return true for a tweet loaded from cache")
	}
	if newStorage.IsProcessed("456") {
		t.Error("IsProcessed() should return false for a new tweet after loading")
	}
}

func TestFileStorage_OldFormatCompatibility(t *testing.T) {
	log := logger.New("DEBUG")
	tempDir := t.TempDir()
	cacheFile := filepath.Join(tempDir, "cache.json")

	// Create a cache file with the old format
	oldFormatJSON := `{"processed_tweets": {"123": true}}`
	if err := os.WriteFile(cacheFile, []byte(oldFormatJSON), 0644); err != nil {
		t.Fatalf("Failed to write old format cache file: %v", err)
	}

	// Test loading from the old format
	storage, err := NewFileStorage(cacheFile, log)
	if err != nil {
		t.Fatalf("NewFileStorage() returned an error when loading old format: %v", err)
	}
	if !storage.IsProcessed("123") {
		t.Error("IsProcessed() should return true for a tweet loaded from old format cache")
	}
}
