package dynalist

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/korjavin/tw2dynalist/internal/logger"
)

func TestAPIClient_AddToInbox_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"_code": "Ok",
			"_msg":  "Item added",
		})
	}))
	defer server.Close()

	log := logger.New("DEBUG")
	client := NewClient("test_token", log)
	client.client = server.Client() // Use the test server's client
	client.BaseURL = server.URL

	err := client.AddToInbox("test content", "test note")
	if err != nil {
		t.Fatalf("AddToInbox() returned an error: %v", err)
	}
}

func TestAPIClient_AddToInbox_RateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK) // Dynalist API returns 200 even for errors
		json.NewEncoder(w).Encode(map[string]string{
			"_code": "TooManyRequests",
			"_msg":  "Rate limit exceeded",
		})
	}))
	defer server.Close()

	log := logger.New("DEBUG")
	client := NewClient("test_token", log)
	client.client = server.Client()
	client.BaseURL = server.URL

	err := client.AddToInbox("test content", "test note")
	if err == nil {
		t.Fatal("AddToInbox() should have returned an error for rate limit")
	}
	if err.Error() != "dynalist rate limit: Rate limit exceeded" {
		t.Errorf("Expected rate limit error, got: %v", err)
	}
}

func TestAPIClient_AddToInbox_InvalidToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"_code": "InvalidToken",
			"_msg":  "Token is invalid",
		})
	}))
	defer server.Close()

	log := logger.New("DEBUG")
	client := NewClient("test_token", log)
	client.client = server.Client()
	client.BaseURL = server.URL

	err := client.AddToInbox("test content", "test note")
	if err == nil {
		t.Fatal("AddToInbox() should have returned an error for invalid token")
	}
	if err.Error() != "dynalist invalid token: Token is invalid" {
		t.Errorf("Expected invalid token error, got: %v", err)
	}
}
