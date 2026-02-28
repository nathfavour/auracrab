package immune

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/nathfavour/auracrab/pkg/config"
	"github.com/nathfavour/auracrab/pkg/vault"
)

type SwarmMessage struct {
	From      int       `json:"from"`
	To        int       `json:"to"` // 0 for broadcast
	Type      string    `json:"type"`
	Payload   any       `json:"payload"`
	Timestamp time.Time `json:"timestamp"`
}

type SwarmBus struct {
	inbox string
	self  int
}

func NewSwarmBus(selfPID int) *SwarmBus {
	inboxDir := filepath.Join(config.DataDir(), "swarm", "inbox")
	_ = os.MkdirAll(inboxDir, 0755)
	return &SwarmBus{
		inbox: inboxDir,
		self:  selfPID,
	}
}

// Broadcast sends an encrypted message to all nodes.
func (sb *SwarmBus) Broadcast(msgType string, payload any) error {
	return sb.Send(0, msgType, payload)
}

// Send sends an encrypted message to a specific node (or broadcast if to=0).
func (sb *SwarmBus) Send(to int, msgType string, payload any) error {
	msg := SwarmMessage{
		From:      sb.self,
		To:        to,
		Type:      msgType,
		Payload:   payload,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	encrypted, err := vault.Encrypt(data)
	if err != nil {
		return err
	}

	// Message file format: msg_<timestamp>_<from>_<to>.enc
	fileName := fmt.Sprintf("msg_%d_%d_%d.enc", time.Now().UnixNano(), sb.self, to)
	path := filepath.Join(sb.inbox, fileName)

	return os.WriteFile(path, encrypted, 0600)
}

// Listen reads and decrypts messages from the inbox.
func (sb *SwarmBus) Listen() ([]SwarmMessage, error) {
	files, err := filepath.Glob(filepath.Join(sb.inbox, "msg_*.enc"))
	if err != nil {
		return nil, err
	}

	var messages []SwarmMessage
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}

		decrypted, err := vault.Decrypt(data)
		if err != nil {
			continue
		}

		var msg SwarmMessage
		if err := json.Unmarshal(decrypted, &msg); err != nil {
			continue
		}

		// Only process messages for us or broadcasts, and not from us
		if msg.From != sb.self && (msg.To == 0 || msg.To == sb.self) {
			messages = append(messages, msg)
		}

		// Cleanup old messages (older than 1 minute) to avoid inbox bloat
		if time.Since(msg.Timestamp) > 1*time.Minute {
			_ = os.Remove(f)
		}
	}

	return messages, nil
}
