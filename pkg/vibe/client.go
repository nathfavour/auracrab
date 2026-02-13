package vibe

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
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
	conn       net.Conn
	mu         sync.Mutex
}

func NewClient() *Client {
	home, _ := os.UserHomeDir()
	return &Client{
		socketPath: filepath.Join(home, ".vibeauracle", "vibeaura.sock"),
	}
}

func (c *Client) getConn() (net.Conn, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return c.conn, nil
	}

	fmt.Printf("VibeClient: Dialing %s\n", c.socketPath)
	var conn net.Conn
	var err error
	maxRetries := 2
	for i := 0; i < maxRetries; i++ {
		conn, err = net.DialTimeout("unix", c.socketPath, 2*time.Second)
		if err == nil {
			break
		}
		fmt.Printf("VibeClient: Dial attempt %d failed: %v\n", i+1, err)
		if i < maxRetries-1 {
			time.Sleep(200 * time.Millisecond)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to vibeauracle UDS: %w", err)
	}

	fmt.Println("VibeClient: Connected to UDS")
	c.conn = conn
	return c.conn, nil
}

func (c *Client) closeConn() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}

func (c *Client) call(method string, payload interface{}) (json.RawMessage, error) {
	fmt.Printf("VibeClient: Calling %s\n", method)
	conn, err := c.getConn()
	if err != nil {
		return nil, err
	}

	// Set a deadline for the call
	conn.SetDeadline(time.Now().Add(60 * time.Second))
	defer conn.SetDeadline(time.Time{})

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
		c.closeConn()
		return nil, fmt.Errorf("failed to write to UDS: %w", err)
	}

	// Wait for response
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	if scanner.Scan() {
		fmt.Printf("VibeClient: Received response for %s\n", method)
		var resp Response
		if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w. raw: %s", err, string(scanner.Bytes()))
		}
		if resp.ID != reqID {
			return nil, fmt.Errorf("response ID mismatch: expected %s, got %s", reqID, resp.ID)
		}
		if resp.Type == "error" {
			return nil, fmt.Errorf("vibeauracle error: %s", string(resp.Payload))
		}
		return resp.Payload, nil
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("VibeClient: Scanner error: %v\n", err)
		c.closeConn()
		return nil, err
	}

	return nil, fmt.Errorf("no response from vibeauracle")
}

func (c *Client) callStream(method string, payload interface{}, onResponse func(json.RawMessage) error) error {
	// Use DialTimeout
	conn, err := net.DialTimeout("unix", c.socketPath, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to vibeauracle UDS: %w", err)
	}
	defer conn.Close()

	// Set a generous deadline for streaming
	conn.SetDeadline(time.Now().Add(5 * time.Minute))

	reqID := fmt.Sprintf("auracrab-%d", time.Now().UnixNano())
	req := Request{
		Type:    "request",
		Method:  method,
		ID:      reqID,
		Payload: payload,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	_, err = conn.Write(append(data, '\n'))
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	for scanner.Scan() {
		var resp Response
		if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
		if resp.ID != reqID {
			continue // Might be a broadcast or other message
		}
		if resp.Type == "error" {
			return fmt.Errorf("vibeauracle error: %s", string(resp.Payload))
		}
		
		if err := onResponse(resp.Payload); err != nil {
			return err
		}

		if resp.Type == "final" {
			break
		}
	}

	return scanner.Err()
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
		Content   string `json:"content"`
		Reasoning string `json:"reasoning,omitempty"`
		Thought   string `json:"thought,omitempty"`
	}
	// Try to unmarshal into a struct with 'content'
	if err := json.Unmarshal(raw, &result); err == nil {
		if result.Content != "" {
			return result.Content, nil
		}
		if result.Reasoning != "" {
			return result.Reasoning, nil
		}
		if result.Thought != "" {
			return result.Thought, nil
		}
	}

	// Fallback to raw string if it's just a string or other JSON
	var str string
	if err := json.Unmarshal(raw, &str); err == nil && str != "" {
		return str, nil
	}

	return string(raw), nil
}

func (c *Client) QueryStream(content string, intent string) (<-chan string, error) {
	payload := QueryPayload{
		Content: content,
		Intent:  intent,
	}
	
	out := make(chan string)
	go func() {
		defer close(out)
		err := c.callStream("query", payload, func(raw json.RawMessage) error {
			var result struct {
				Content string `json:"content"`
				Delta   string `json:"delta"`
			}
			if err := json.Unmarshal(raw, &result); err == nil {
				if result.Delta != "" {
					out <- result.Delta
				} else if result.Content != "" {
					out <- result.Content
				}
			}
			return nil
		})
		if err != nil {
			out <- fmt.Sprintf("\n[Stream Error: %v]", err)
		}
	}()

	return out, nil
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
