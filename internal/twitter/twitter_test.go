package twitter

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/korjavin/tw2dynalist/internal/config"
	"github.com/korjavin/tw2dynalist/internal/logger"
	"golang.org/x/oauth2"

	twitterv2 "github.com/g8rswimmer/go-twitter/v2"
)

// mockAuthorizer is a mock implementation of the twitterv2.Authorizer interface.
type mockAuthorizer struct{}

func (m *mockAuthorizer) Add(req *http.Request) {}

// mockTokenProvider is a mock implementation of the TokenProvider interface.
type mockTokenProvider struct{}

func (m *mockTokenProvider) Token() *oauth2.Token {
	return &oauth2.Token{
		AccessToken: "test_access_token",
	}
}

// mockStorage is a mock implementation of the Storage interface.
type mockStorage struct {
	processedTweets map[string]bool
}

func (m *mockStorage) MarkProcessed(tweetID string) {
	m.processedTweets[tweetID] = true
}

func (m *mockStorage) IsProcessed(tweetID string) bool {
	return m.processedTweets[tweetID]
}

func (m *mockStorage) Save() error {
	return nil
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		processedTweets: make(map[string]bool),
	}
}

func TestAPIClient_GetBookmarks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/2/users/test_user_id/bookmarks" {
			t.Errorf("Expected to request '/2/users/test_user_id/bookmarks', got '%s'", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"data":[{"id":"123","text":"test tweet"}],"includes":{"users":[{"id":"456","username":"testuser"}]}}`)
	}))
	defer server.Close()

	log := logger.New("DEBUG")
	cfg := &config.Config{}
	client := &APIClient{
		client: &twitterv2.Client{
			Authorizer: &mockAuthorizer{},
			Client:     server.Client(),
			Host:       server.URL,
		},
		userID:        "test_user_id",
		logger:        log,
		config:        cfg,
		tokenProvider: &mockTokenProvider{},
	}

	tweets, err := client.GetBookmarks()
	if err != nil {
		t.Fatalf("GetBookmarks() returned an error: %v", err)
	}

	if len(tweets) != 1 {
		t.Fatalf("Expected 1 tweet, got %d", len(tweets))
	}
	if tweets[0].ID != "123" {
		t.Errorf("Expected tweet ID '123', got '%s'", tweets[0].ID)
	}
	if tweets[0].Text != "test tweet" {
		t.Errorf("Expected tweet text 'test tweet', got '%s'", tweets[0].Text)
	}
}

func TestAPIClient_RemoveBookmark(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected 'DELETE' request, got '%s'", r.Method)
		}
		if r.URL.Path != "/2/users/test_user_id/bookmarks/123" {
			t.Errorf("Expected to request '/2/users/test_user_id/bookmarks/123', got '%s'", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"data":{"bookmarked":false}}`)
	}))
	defer server.Close()

	log := logger.New("DEBUG")
	cfg := &config.Config{}
	client := &APIClient{
		client: &twitterv2.Client{
			Authorizer: &mockAuthorizer{},
			Client:     server.Client(),
			Host:       server.URL,
		},
		userID:        "test_user_id",
		logger:        log,
		config:        cfg,
		tokenProvider: &mockTokenProvider{},
	}

	err := client.RemoveBookmark("123")
	if err != nil {
		t.Fatalf("RemoveBookmark() returned an error: %v", err)
	}
}

func TestAPIClient_CleanupProcessedBookmarks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, `{"data":[{"id":"123","text":"processed tweet"},{"id":"456","text":"unprocessed tweet"}]}`)
		} else if r.Method == "DELETE" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"data":{"bookmarked":false}}`)
		}
	}))
	defer server.Close()

	log := logger.New("DEBUG")
	cfg := &config.Config{}
	storage := newMockStorage()
	storage.MarkProcessed("123")

	client := &APIClient{
		client: &twitterv2.Client{
			Authorizer: &mockAuthorizer{},
			Client:     server.Client(),
			Host:       server.URL,
		},
		userID:        "test_user_id",
		logger:        log,
		config:        cfg,
		tokenProvider: &mockTokenProvider{},
	}

	err := client.CleanupProcessedBookmarks(storage)
	if err != nil {
		t.Fatalf("CleanupProcessedBookmarks() returned an error: %v", err)
	}
}
