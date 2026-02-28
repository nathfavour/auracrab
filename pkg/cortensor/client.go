package cortensor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SessionMetadata stores Cortensor-specific session state
type SessionMetadata struct {
	SessionID     string    `json:"session_id"`
	NodeID        string    `json:"node_id"`
	Expiry        time.Time `json:"expiry"`
	CORBalance    float64   `json:"cor_balance"`
	RouterEndpoint string    `json:"router_endpoint"`
}

// Client handles communication with the Cortensor Protocol
type Client struct {
	routerEndpoint string
	sessionID      string
	httpClient     *http.Client
}

// CompletionRequest matches the expected payload for the Cortensor router
type RouterCompletionRequest struct {
	Content       string            `json:"content"`
	TaskType      string            `json:"task_type"`
	MinerSelector string            `json:"miner_selector,omitempty"`
	Context       map[string]string `json:"context,omitempty"`
	SessionID     string            `json:"session_id"`
}

// RouterCompletionResponse is the response structure from the router
type RouterCompletionResponse struct {
	Content   string `json:"content"`
	Reasoning string `json:"reasoning,omitempty"`
	ProofHash string `json:"proof_hash,omitempty"`
	MinerID   string `json:"miner_id"`
}

func NewClient(endpoint, sessionID string) *Client {
	return &Client{
		routerEndpoint: endpoint,
		sessionID:      sessionID,
		httpClient: &http.Client{
			Timeout: 90 * time.Second,
		},
	}
}

// Handshake initializes a session with the Cortensor Router
func (c *Client) Handshake(ctx context.Context) (*SessionMetadata, error) {
	// Implementation for Cortensor session handshake
	// For now, this is a placeholder for the REST call to /session/init
	url := fmt.Sprintf("%s/v1/session/init", c.routerEndpoint)
	
	reqBody, _ := json.Marshal(map[string]string{
		"session_id": c.sessionID,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	// In a real implementation, we would call the router.
	// For the prototype, we simulate a successful handshake if the endpoint is set.
	if c.routerEndpoint == "" {
		return nil, fmt.Errorf("cortensor router endpoint not configured")
	}

	// Simulated response
	return &SessionMetadata{
		SessionID:      c.sessionID,
		NodeID:         "router-alpha-1",
		Expiry:         time.Now().Add(24 * time.Hour),
		CORBalance:     100.0,
		RouterEndpoint: c.routerEndpoint,
	}, nil
}

// Query sends an inference request to the Cortensor network
func (c *Client) Query(ctx context.Context, content, intent string) (*RouterCompletionResponse, error) {
	url := fmt.Sprintf("%s/v1/inference/completion", c.routerEndpoint)

	taskType := "general"
	if intent == "agent" || intent == "plan" {
		taskType = "code_generation"
	}

	routerReq := RouterCompletionRequest{
		Content:   content,
		TaskType:  taskType,
		SessionID: c.sessionID,
		Context: map[string]string{
			"agent": "auracrab",
		},
	}

	reqBody, err := json.Marshal(routerReq)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cortensor router call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cortensor router returned status: %d", resp.StatusCode)
	}

	var routerResp RouterCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&routerResp); err != nil {
		return nil, fmt.Errorf("failed to decode cortensor response: %w", err)
	}

	return &routerResp, nil
}
