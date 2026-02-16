package connect

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type BrowserClient struct {
	Conn      *websocket.Conn
	UserAgent string
	Profile   string
	WindowID  string
	Connected time.Time
}

type BrowserChannel struct {
	mu          sync.RWMutex
	clients     map[*websocket.Conn]*BrowserClient
	onMessage   func(platform string, chatID string, from string, text string) string
	upgrader    websocket.Upgrader
	port        int
	pendingResp sync.Map // map[string]chan string
}

func GetBrowserChannel() *BrowserChannel {
	return globalBrowserChannel
}

func NewBrowserChannel(port int) *BrowserChannel {
	globalBrowserChannel = &BrowserChannel{
		clients: make(map[*websocket.Conn]*BrowserClient),
		port:    port,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
	return globalBrowserChannel
}

func (c *BrowserChannel) IsActive() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.clients) > 0
}

func (c *BrowserChannel) Name() string {
	return "browser"
}

func (c *BrowserChannel) Start(ctx context.Context, onMessage func(platform string, chatID string, from string, text string) string) error {
	c.onMessage = onMessage

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", c.handleWebSocket)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", c.port),
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		server.Close()
	}()

	fmt.Printf("Browser: WebSocket server starting on %s\n", server.Addr)
	// This will block, so we run it in a goroutine if we want to return from Start
	// But Start is usually called in a goroutine anyway in butler.go
	return server.ListenAndServe()
}

func (c *BrowserChannel) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := c.upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("Browser: Upgrade error: %v\n", err)
		return
	}
	defer conn.Close()

	client := &BrowserClient{
		Conn:      conn,
		Connected: time.Now(),
		UserAgent: r.Header.Get("User-Agent"),
	}

	c.mu.Lock()
	c.clients[conn] = client
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.clients, conn)
		c.mu.Unlock()
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var msg struct {
			Type    string `json:"type"`
			Content string `json:"content"`
			ID      string `json:"id"`
			Profile string `json:"profile,omitempty"`
		}
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		if msg.Type == "register" {
			c.mu.Lock()
			client.Profile = msg.Profile
			c.mu.Unlock()
			fmt.Printf("Browser: Registered profile %s\n", msg.Profile)
			continue
		}

		if msg.Type == "response" && msg.ID != "" {
			if ch, ok := c.pendingResp.Load(msg.ID); ok {
				ch.(chan string) <- msg.Content
				continue
			}
		}

		if c.onMessage != nil {
			// Map incoming browser events to butler messages
			// Using "browser" as platform and "active" or a specific ID as chatID
			reply := c.onMessage("browser", "active", "extension", msg.Content)
			if reply != "" {
				c.Send("active", reply)
			}
		}
	}
}

func (c *BrowserChannel) Request(ctx context.Context, command string) (string, error) {
	if !c.IsActive() {
		return "", fmt.Errorf("no browser extension connected")
	}

	id := fmt.Sprintf("req_%d", time.Now().UnixNano())
	ch := make(chan string, 1)
	c.pendingResp.Store(id, ch)
	defer c.pendingResp.Delete(id)

	payload, _ := json.Marshal(map[string]string{
		"type":    "command",
		"content": command,
		"id":      id,
	})

	c.mu.RLock()
	for conn := range c.clients {
		_ = conn.WriteMessage(websocket.TextMessage, payload)
	}
	c.mu.RUnlock()

	select {
	case resp := <-ch:
		return resp, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (c *BrowserChannel) Stop() error {
	return nil
}

func (c *BrowserChannel) Send(to string, text string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	msg, _ := json.Marshal(map[string]string{
		"type":    "command",
		"content": text,
	})

	for conn := range c.clients {
		err := conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			fmt.Printf("Browser: Write error: %v\n", err)
		}
	}
	return nil
}

func (c *BrowserChannel) Broadcast(message string) error {
	return c.Send("", message)
}
