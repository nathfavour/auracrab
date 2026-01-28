package memory

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/nathfavour/auracrab/pkg/config"
	_ "modernc.org/sqlite"
)

// Message represents a single message in a conversation.
type Message struct {
	ID        int64     `json:"id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// Conversation represents a thread of messages.
type Conversation struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// HistoryStore handles persistent conversation history using SQLite.
type HistoryStore struct {
	db *sql.DB
}

// NewHistoryStore initializes the SQLite database and returns a HistoryStore.
func NewHistoryStore() (*HistoryStore, error) {
	dbPath := filepath.Join(config.DataDir(), "history.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open history database: %v", err)
	}

	// Create tables if they don't exist
	query := `
	CREATE TABLE IF NOT EXISTS conversations (
		id TEXT PRIMARY KEY,
		title TEXT,
		created_at DATETIME,
		updated_at DATETIME
	);
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		conversation_id TEXT,
		role TEXT,
		content TEXT,
		timestamp DATETIME,
		FOREIGN KEY(conversation_id) REFERENCES conversations(id)
	);
	CREATE TABLE IF NOT EXISTS platform_mappings (
		platform TEXT,
		platform_id TEXT,
		conversation_id TEXT,
		PRIMARY KEY(platform, platform_id),
		FOREIGN KEY(conversation_id) REFERENCES conversations(id)
	);
	CREATE TABLE IF NOT EXISTS authorized_entities (
		platform TEXT,
		platform_id TEXT,
		authorized_at DATETIME,
		PRIMARY KEY(platform, platform_id)
	);
	`
	if _, err := db.Exec(query); err != nil {
		return nil, fmt.Errorf("failed to initialize history tables: %v", err)
	}

	return &HistoryStore{db: db}, nil
}

// IsAuthorized checks if a platform ID is authorized to interact with the bot.
func (h *HistoryStore) IsAuthorized(platform, platformID string) (bool, error) {
	var exists bool
	err := h.db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM authorized_entities WHERE platform = ? AND platform_id = ?)",
		platform, platformID,
	).Scan(&exists)
	return exists, err
}

// AuthorizeEntity marks a platform ID as authorized.
func (h *HistoryStore) AuthorizeEntity(platform, platformID string) error {
	_, err := h.db.Exec(
		"INSERT OR REPLACE INTO authorized_entities (platform, platform_id, authorized_at) VALUES (?, ?, ?)",
		platform, platformID, time.Now(),
	)
	return err
}

// DeauthorizeEntity removes authorization for a platform ID.
func (h *HistoryStore) DeauthorizeEntity(platform, platformID string) error {
	_, err := h.db.Exec(
		"DELETE FROM authorized_entities WHERE platform = ? AND platform_id = ?",
		platform, platformID,
	)
	return err
}

// GetOrCreateConversationForPlatform retrieves an existing conversation UUID or creates a new one for a platform (e.g., "telegram") and platform-specific ID (e.g., chatID).
func (h *HistoryStore) GetOrCreateConversationForPlatform(platform, platformID string) (string, error) {
	var convID string
	err := h.db.QueryRow(
		"SELECT conversation_id FROM platform_mappings WHERE platform = ? AND platform_id = ?",
		platform, platformID,
	).Scan(&convID)

	if err == sql.ErrNoRows {
		// Create new conversation
		title := fmt.Sprintf("%s Conversation (%s)", platform, platformID)
		convID, err = h.CreateConversation(title)
		if err != nil {
			return "", err
		}

		// Create mapping
		_, err = h.db.Exec(
			"INSERT INTO platform_mappings (platform, platform_id, conversation_id) VALUES (?, ?, ?)",
			platform, platformID, convID,
		)
		if err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}

	return convID, nil
}

// Close closes the database connection.
func (h *HistoryStore) Close() error {
	return h.db.Close()
}

// CreateConversation creates a new conversation with a UUID and returns the ID.
func (h *HistoryStore) CreateConversation(title string) (string, error) {
	id := uuid.New().String()
	now := time.Now()
	_, err := h.db.Exec(
		"INSERT INTO conversations (id, title, created_at, updated_at) VALUES (?, ?, ?, ?)",
		id, title, now, now,
	)
	if err != nil {
		return "", err
	}
	return id, nil
}

// AddMessage adds a message to a specific conversation.
func (h *HistoryStore) AddMessage(convID, role, content string) error {
	now := time.Now()
	_, err := h.db.Exec(
		"INSERT INTO messages (conversation_id, role, content, timestamp) VALUES (?, ?, ?, ?)",
		convID, role, content, now,
	)
	if err != nil {
		return err
	}

	// Update the conversation's updated_at timestamp
	_, err = h.db.Exec("UPDATE conversations SET updated_at = ? WHERE id = ?", now, convID)
	return err
}

// GetHistory retrieves all messages for a conversation, ordered by timestamp.
func (h *HistoryStore) GetHistory(convID string) ([]Message, error) {
	rows, err := h.db.Query(
		"SELECT id, role, content, timestamp FROM messages WHERE conversation_id = ? ORDER BY timestamp ASC",
		convID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.Role, &m.Content, &m.Timestamp); err != nil {
			return nil, err
		}
		history = append(history, m)
	}
	return history, nil
}

// ListConversations returns a list of all conversations, newest first.
func (h *HistoryStore) ListConversations() ([]Conversation, error) {
	rows, err := h.db.Query("SELECT id, title, created_at, updated_at FROM conversations ORDER BY updated_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conversations []Conversation
	for rows.Next() {
		var c Conversation
		if err := rows.Scan(&c.ID, &c.Title, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		conversations = append(conversations, c)
	}
	return conversations, nil
}

// DeleteConversation removes a conversation and all its messages.
func (h *HistoryStore) DeleteConversation(convID string) error {
	tx, err := h.db.Begin()
	if err != nil {
		return err
	}
	
	_, err = tx.Exec("DELETE FROM messages WHERE conversation_id = ?", convID)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec("DELETE FROM conversations WHERE id = ?", convID)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}
