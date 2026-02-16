package social

import (
	"context"
	"fmt"
	"strings"

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

				var update Update
				if tgUpdate.Message != nil {
					from := fmt.Sprintf("@%s", tgUpdate.Message.From.UserName)
					if tgUpdate.Message.From.UserName == "" {
						from = fmt.Sprintf("%d", tgUpdate.Message.From.ID)
					}
					update = Update{
						ChatID:  fmt.Sprintf("%d", tgUpdate.Message.Chat.ID),
						Text:    tgUpdate.Message.Text,
						RawFrom: from,
					}
				} else if tgUpdate.CallbackQuery != nil {
					// Handle callback as a text command for simplicity in BotManager
					from := fmt.Sprintf("@%s", tgUpdate.CallbackQuery.From.UserName)
					if tgUpdate.CallbackQuery.From.UserName == "" {
						from = fmt.Sprintf("%d", tgUpdate.CallbackQuery.From.ID)
					}

					// Prepend / for callback data to treat them as commands
					text := tgUpdate.CallbackQuery.Data
					if !strings.HasPrefix(text, "/") {
						text = "/" + text
					}

					update = Update{
						ChatID:  fmt.Sprintf("%d", tgUpdate.CallbackQuery.Message.Chat.ID),
						Text:    text,
						RawFrom: from,
					}

					// Answer callback query to stop loading spinner
					_, _ = p.bot.Request(tgbotapi.NewCallback(tgUpdate.CallbackQuery.ID, ""))
				} else {
					continue
				}

				updates <- update
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
	} else if options.ParseMode == ParseModeMarkdown {
		msg.ParseMode = "Markdown"
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
		tgbotapi.NewKeyboardButton("/pay"),
		tgbotapi.NewKeyboardButton("/wallet"),
		tgbotapi.NewKeyboardButton("/status"),
	),
)

func NewModeInlineKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💬 Chat", "mode_chat"),
			tgbotapi.NewInlineKeyboardButtonData("🤖 Agent", "mode_agent"),
			tgbotapi.NewInlineKeyboardButtonData("🐚 Shell", "mode_shell"),
		),
	)
}

func NewPaymentKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰 Quick Pay 1 USDC", "pay_1_usdc"),
			tgbotapi.NewInlineKeyboardButtonData("🏦 Wallet Info", "wallet_info"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Settle Pending", "settle_all"),
		),
	)
}
