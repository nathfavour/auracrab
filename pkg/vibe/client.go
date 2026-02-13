package vibe

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

type Request struct {
	Type    string      `json:"type"`
	Method  string      `json:"method"`
	ID      string      `json:"id"`
	Payload interface{} `json:"payload"`
}

type Response struct {
	Type    string          `json:"type"`
	ID      string          `json:"id"`
	Payload json.RawMessage `json:"payload"`
}

type QueryPayload struct {
	Content string `json:"content"`
	Intent  string `json:"intent,omitempty"`
}

type Client struct {
	socketPath string
}

func NewClient() *Client {
	home, _ := os.UserHomeDir()
	return &Client{
		socketPath: filepath.Join(home, ".vibeauracle", "vibeaura.sock"),
	}
}

func (c *Client) call(method string, payload interface{}) (json.RawMessage, error) {
	var conn net.Conn
	var err error

	// Retry logic for connection
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		conn, err = net.Dial("unix", c.socketPath)
		if err == nil {
			break
		}
		if i < maxRetries-1 {
			time.Sleep(500 * time.Millisecond)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to vibeauracle UDS after retries: %w", err)
	}
	defer conn.Close()

	reqID := fmt.Sprintf("auracrab-%d", time.Now().UnixNano())
	req := Request{
		Type:    "request",
		Method:  method,
		ID:      reqID,
		Payload: payload,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	_, err = conn.Write(append(data, '\n'))
	if err != nil {
		return nil, err
	}

	// Wait for response
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	if scanner.Scan() {
		var resp Response
		if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w. raw: %s", err, string(scanner.Bytes()))
		}
		if resp.ID != reqID {
			return nil, fmt.Errorf("response ID mismatch: expected %s, got %s", reqID, resp.ID)
		}
		if resp.Type == "error" {
			var errPayload struct {
				Message string `json:"message"`
			}
			if err := json.Unmarshal(resp.Payload, &errPayload); err == nil && errPayload.Message != "" {
				return nil, fmt.Errorf("vibeauracle error: %s", errPayload.Message)
			}
			return nil, fmt.Errorf("vibeauracle error: %s", string(resp.Payload))
		}
		return resp.Payload, nil
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return nil, fmt.Errorf("no response from vibeauracle")
}

func (c *Client) Query(content string, intent string) (string, error) {
	payload := QueryPayload{
		Content: content,
		Intent:  intent,
	}
	raw, err := c.call("query", payload)
	if err != nil {
		return "", err
	}

	var result struct {
		Content string `json:"content"`
	}
	// Try to unmarshal into a struct with 'content'
	if err := json.Unmarshal(raw, &result); err == nil && result.Content != "" {
		return result.Content, nil
	}

	// Fallback to raw string if it's just a string or other JSON
	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		return str, nil
	}

	return string(raw), nil
}

func (c *Client) Embed(content string) ([]float64, error) {
	raw, err := c.call("embed", map[string]string{"content": content})
	if err != nil {
		return nil, err
	}

	var result []float64
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *Client) Ping() error {
	_, err := c.call("ping", map[string]interface{}{})
	return err
}

func (c *Client) GetStatus() (json.RawMessage, error) {
	return c.call("status", map[string]interface{}{})
}
