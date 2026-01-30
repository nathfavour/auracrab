package social

import (
	"context"
)

type Update struct {
	ChatID  string
	Text    string
	RawFrom interface{}
}

type BotCommand struct {
	Text        string
	Description string
}

type MessageOptions struct {
	ParseMode string
	Keyboard  interface{}
}

const (
	ParseModeHTML = "HTML"
	ActionTyping  = "typing"
)

type MessengerProvider interface {
	GetName() string
	GetUpdates(ctx context.Context) (<-chan Update, error)
	SendMessage(chatID string, text string, options MessageOptions) error
	SendAction(chatID string, action string) error
	SetCommands(commands []BotCommand) error
}
