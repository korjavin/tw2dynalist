package dynalist

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/korjavin/tw2dynalist/internal/logger"
)

// Client defines the interface for interacting with the Dynalist API.
type Client interface {
	AddToInbox(content, note string) error
}

// APIClient implements the Client interface for the Dynalist API.
type APIClient struct {
	token   string
	client  *http.Client
	logger  *logger.Logger
	BaseURL string
}

// InboxRequest represents the request to add an item to Dynalist inbox.
type InboxRequest struct {
	Token   string `json:"token"`
	Content string `json:"content"`
	Note    string `json:"note,omitempty"`
}

// NewClient creates a new Dynalist API client.
func NewClient(token string, logger *logger.Logger) *APIClient {
	logger.Debug("Creating new Dynalist client")
	return &APIClient{
		token:   token,
		client:  &http.Client{Timeout: 10 * time.Second},
		logger:  logger,
		BaseURL: "https://dynalist.io/api/v1/inbox/add",
	}
}

// AddToInbox adds an item to the Dynalist inbox.
func (c *APIClient) AddToInbox(content, note string) error {
	c.logger.Debug("Preparing request to add item to Dynalist inbox")
	reqBody := InboxRequest{
		Token:   c.token,
		Content: content,
		Note:    note,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	c.logger.Debug("Sending request to Dynalist API at %s", c.BaseURL)
	req, err := http.NewRequest("POST", c.BaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
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

	c.logger.Debug("Dynalist API response: %v", result)

	if result["_code"] != "Ok" {
		code := result["_code"]
		msg := result["_msg"]

		switch code {
		case "TooManyRequests":
			c.logger.Warn("Dynalist rate limit hit, pausing for 2 seconds")
			time.Sleep(2 * time.Second)
			return fmt.Errorf("dynalist rate limit: %s", msg)
		case "InvalidToken":
			c.logger.Error("Dynalist token is invalid: %s", msg)
			return fmt.Errorf("dynalist invalid token: %s", msg)
		case "Unauthorized":
			c.logger.Error("Dynalist unauthorized: %s", msg)
			return fmt.Errorf("dynalist unauthorized: %s", msg)
		default:
			c.logger.Error("Dynalist API error [%s]: %s", code, msg)
			return fmt.Errorf("dynalist API error [%s]: %s", code, msg)
		}
	}

	c.logger.Debug("Successfully added item to Dynalist inbox")
	return nil
}
