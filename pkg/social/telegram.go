package social

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TelegramProvider struct {
	bot *tgbotapi.BotAPI
}

func NewTelegramProvider(token string) (*TelegramProvider, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}
	return &TelegramProvider{bot: bot}, nil
}

func (p *TelegramProvider) GetName() string {
	return p.bot.Self.UserName
}

func (p *TelegramProvider) GetUpdates(ctx context.Context) (<-chan Update, error) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	tgUpdates := p.bot.GetUpdatesChan(u)

	updates := make(chan Update)
	go func() {
		defer close(updates)
		for {
			select {
			case <-ctx.Done():
				return
			case tgUpdate, ok := <-tgUpdates:
				if !ok {
					return
				}
				if tgUpdate.Message == nil {
					continue
				}

				from := fmt.Sprintf("@%s", tgUpdate.Message.From.UserName)
				if tgUpdate.Message.From.UserName == "" {
					from = fmt.Sprintf("%d", tgUpdate.Message.From.ID)
				}

				updates <- Update{
					ChatID:  fmt.Sprintf("%d", tgUpdate.Message.Chat.ID),
					Text:    tgUpdate.Message.Text,
					RawFrom: from,
				}
			}
		}
	}()

	return updates, nil
}

func (p *TelegramProvider) SendMessage(chatID string, text string, options MessageOptions) error {
	var id int64
	fmt.Sscanf(chatID, "%d", &id)
	msg := tgbotapi.NewMessage(id, text)
	if options.ParseMode == ParseModeHTML {
		msg.ParseMode = "HTML"
	}
	if options.Keyboard != nil {
		msg.ReplyMarkup = options.Keyboard
	}
	_, err := p.bot.Send(msg)
	return err
}

func (p *TelegramProvider) SendAction(chatID string, action string) error {
	var id int64
	fmt.Sscanf(chatID, "%d", &id)
	tgAction := ""
	switch action {
	case ActionTyping:
		tgAction = tgbotapi.ChatTyping
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
	_, err := p.bot.Send(tgbotapi.NewChatAction(id, tgAction))
	return err
}

func (p *TelegramProvider) SetCommands(commands []BotCommand) error {
	tgCommands := make([]tgbotapi.BotCommand, len(commands))
	for i, c := range commands {
		tgCommands[i] = tgbotapi.BotCommand{
			Command:     c.Text,
			Description: c.Description,
		}
	}
	_, err := p.bot.Request(tgbotapi.NewSetMyCommands(tgCommands...))
	return err
}

var TelegramModeKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Mode: Chat"),
		tgbotapi.NewKeyboardButton("Mode: Agent"),
		tgbotapi.NewKeyboardButton("Mode: Shell"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("/status"),
		tgbotapi.NewKeyboardButton("/help"),
	),
)
