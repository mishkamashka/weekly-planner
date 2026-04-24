package bot

import (
	"context"
	"log/slog"
	"strings"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (b *Bot) handlePing(ctx context.Context, tg *tgbot.Bot, update *models.Update) {
	tg.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "pong",
	})
}

func (b *Bot) handleStart(ctx context.Context, tg *tgbot.Bot, update *models.Update) {
	from := update.Message.From
	if _, err := b.store.GetOrCreateUser(from.ID, from.FirstName); err != nil {
		slog.Error("getOrCreateUser failed", "err", err)
	}

	tg.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        "Hey! I'm your weekly planner bot.\n\nTap ➕ to add tasks to your backlog.",
		ReplyMarkup: mainKeyboard(),
	})
}

func (b *Bot) handleAddTaskPrompt(ctx context.Context, tg *tgbot.Bot, update *models.Update) {
	b.setState(update.Message.From.ID, stateWaitingForTask)
	tg.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        "What's the task?",
		ReplyMarkup: &models.ReplyKeyboardRemove{RemoveKeyboard: true},
	})
}

func (b *Bot) handleAddCommand(ctx context.Context, tg *tgbot.Bot, update *models.Update) {
	title := strings.TrimSpace(strings.TrimPrefix(update.Message.Text, "/add"))
	if title == "" {
		b.setState(update.Message.From.ID, stateWaitingForTask)
		tg.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID:      update.Message.Chat.ID,
			Text:        "What's the task?",
			ReplyMarkup: &models.ReplyKeyboardRemove{RemoveKeyboard: true},
		})
		return
	}
	b.addTask(ctx, tg, update.Message.Chat.ID, update.Message.From.ID, title)
}

func (b *Bot) handleDefault(ctx context.Context, tg *tgbot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	if b.getState(update.Message.From.ID) != stateWaitingForTask {
		tg.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID:      update.Message.Chat.ID,
			Text:        "Tap ➕ Add task to add something to your backlog.",
			ReplyMarkup: mainKeyboard(),
		})
		return
	}
	title := strings.TrimSpace(update.Message.Text)
	if title == "" {
		return
	}
	b.setState(update.Message.From.ID, stateIdle)
	b.addTask(ctx, tg, update.Message.Chat.ID, update.Message.From.ID, title)
}

func (b *Bot) addTask(ctx context.Context, tg *tgbot.Bot, chatID, telegramID int64, title string) {
	user, err := b.store.GetOrCreateUser(telegramID, "")
	if err != nil {
		slog.Error("getOrCreateUser failed", "err", err)
		return
	}
	if _, err := b.store.AddTask(user.ID, title); err != nil {
		slog.Error("addTask failed", "err", err)
		return
	}
	tg.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:      chatID,
		Text:        "Added to backlog: " + title,
		ReplyMarkup: mainKeyboard(),
	})
}
