package connect

import (
	"context"
)

// Channel defines an integration surface like Telegram, Discord, or Signal.
type Channel interface {
	Name() string
	Start(ctx context.Context, onMessage func(platform string, chatID string, from string, text string) string) error
	Stop() error
	Send(to string, text string) error
	Broadcast(message string) error
}

var channels = make(map[string]Channel)

func RegisterChannel(c Channel) {
	channels[c.Name()] = c
}

func GetChannels() map[string]Channel {
	return channels
}
