package social

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type DiscordProvider struct {
	session *discordgo.Session
}

func NewDiscordProvider(token string) (*DiscordProvider, error) {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}
	return &DiscordProvider{session: dg}, nil
}

func (p *DiscordProvider) GetName() string {
	if p.session.State != nil && p.session.State.User != nil {
		return p.session.State.User.Username
	}
	return "DiscordBot"
}

func (p *DiscordProvider) GetUpdates(ctx context.Context) (<-chan Update, error) {
	updates := make(chan Update)

	p.session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == s.State.User.ID {
			return
		}

		from := fmt.Sprintf("%s#%s", m.Author.Username, m.Author.Discriminator)
		if m.Author.Discriminator == "0" || m.Author.Discriminator == "" {
			from = m.Author.Username
		}

		updates <- Update{
			ChatID:  m.ChannelID,
			Text:    m.Content,
			RawFrom: from,
		}
	})

	err := p.session.Open()
	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		p.session.Close()
		close(updates)
	}()

	return updates, nil
}

func (p *DiscordProvider) SendMessage(chatID string, text string, options MessageOptions) error {
	_, err := p.session.ChannelMessageSend(chatID, text)
	return err
}

func (p *DiscordProvider) SendAction(chatID string, action string) error {
	if action == ActionTyping {
		return p.session.ChannelTyping(chatID)
	}
	return nil
}

func (p *DiscordProvider) SetCommands(commands []BotCommand) error {
	// Discord slash commands are more complex to implement via this simple interface,
	// so we'll skip for now or implement as simple help text.
	return nil
}
