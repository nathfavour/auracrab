package connect

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

// DiscordChannel implements the Discord integration.
type DiscordChannel struct {
	Token   string
	session *discordgo.Session
}

func (d *DiscordChannel) Name() string {
	return "discord"
}

func (d *DiscordChannel) Start(ctx context.Context, onMessage func(from string, text string) string) error {
	dg, err := discordgo.New("Bot " + d.Token)
	if err != nil {
		return fmt.Errorf("error creating Discord session: %v", err)
	}

	d.session = dg

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		// Ignore all messages created by the bot itself
		if m.Author.ID == s.State.User.ID {
			return
		}

		from := fmt.Sprintf("%s#%s", m.Author.Username, m.Author.Discriminator)
		if m.Author.Discriminator == "0" || m.Author.Discriminator == "" {
			from = m.Author.Username
		}

		text := m.Content
		if text == "" {
			return
		}

		log.Printf("[Discord][%s] %s", from, text)

		// Dispatch to Butler
		reply := onMessage(from, text)

		// Send reply
		_, err := s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("<@%s> %s", m.Author.ID, reply))
		if err != nil {
			log.Printf("Error sending Discord message: %v", err)
		}
	})

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		return fmt.Errorf("error opening Discord connection: %v", err)
	}

	log.Printf("Auracrab: Discord bot is now running.")

	go func() {
		<-ctx.Done()
		d.Stop()
	}()

	return nil
}

func (d *DiscordChannel) Stop() error {
	if d.session != nil {
		log.Printf("Auracrab: Closing Discord connection.")
		return d.session.Close()
	}
	return nil
}

func init() {
	// Discord will be registered in init.go if token is present
}
