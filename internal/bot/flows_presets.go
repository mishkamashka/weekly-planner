package bot

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"strconv"
	"strings"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (b *Bot) handlePresets(ctx context.Context, tg *tgbot.Bot, update *models.Update) {
	user, err := b.store.GetOrCreateUser(update.Message.From.ID, "")
	if err != nil {
		slog.Error("getOrCreateUser failed", "err", err)
		return
	}
	b.sendPresetList(ctx, tg, update.Message.Chat.ID, user.ID)
}

func (b *Bot) sendPresetList(ctx context.Context, tg *tgbot.Bot, chatID, userID int64) {
	presets, err := b.store.GetPresets(userID)
	if err != nil {
		slog.Error("getPresets failed", "err", err)
		return
	}
	tg.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:      chatID,
		Text:        presetsText(presets),
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: presetsKeyboard(presets),
	})
}

func (b *Bot) handlePresetCallback(ctx context.Context, tg *tgbot.Bot, update *models.Update) {
	defer tg.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})

	data := update.CallbackQuery.Data
	msg := update.CallbackQuery.Message.Message
	if msg == nil {
		return
	}
	chatID := msg.Chat.ID
	telegramID := update.CallbackQuery.From.ID

	if data == "pr:new" {
		b.setState(telegramID, stateAddingPresetTitle)
		tg.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: chatID,
			Text:   "What's the recurring task title?",
		})
		return
	}

	parts := strings.SplitN(data, ":", 3)
	if len(parts) != 3 {
		return
	}
	action, valStr := parts[1], parts[2]

	user, err := b.store.GetOrCreateUser(telegramID, "")
	if err != nil {
		slog.Error("getOrCreateUser failed", "err", err)
		return
	}

	switch action {
	case "toggle":
		id, err := strconv.ParseInt(valStr, 10, 64)
		if err != nil {
			return
		}
		if err := b.store.TogglePreset(id); err != nil {
			slog.Error("togglePreset failed", "err", err)
			return
		}
		b.refreshPresetList(ctx, tg, chatID, msg.ID, user.ID)

	case "del":
		id, err := strconv.ParseInt(valStr, 10, 64)
		if err != nil {
			return
		}
		if err := b.store.DeletePreset(id); err != nil {
			slog.Error("deletePreset failed", "err", err)
			return
		}
		b.refreshPresetList(ctx, tg, chatID, msg.ID, user.ID)

	case "day":
		if valStr == "done" {
			title := b.popPendingPreset(telegramID)
			days := b.popPendingPresetDays(telegramID)
			if title == "" {
				return
			}
			created := 0
			for d, selected := range days {
				if !selected {
					continue
				}
				if _, err := b.store.AddPreset(user.ID, title, d); err != nil {
					slog.Error("addPreset failed", "err", err)
				} else {
					created++
				}
			}
			if created == 0 {
				return
			}
			b.refreshPresetList(ctx, tg, chatID, msg.ID, user.ID)
		} else {
			n, err := strconv.Atoi(valStr)
			if err != nil || n < 0 || n > 6 {
				return
			}
			b.togglePendingPresetDay(telegramID, n)
			days := b.getPendingPresetDays(telegramID)
			title := b.getPendingPreset(telegramID)
			tg.EditMessageText(ctx, &tgbot.EditMessageTextParams{
				ChatID:    chatID,
				MessageID: msg.ID,
				Text: fmt.Sprintf("Which days should <b>%s</b> repeat?\nTap days to select, then Save.",
					html.EscapeString(title)),
				ParseMode:   models.ParseModeHTML,
				ReplyMarkup: presetDayMultiKeyboard(days),
			})
		}
	}
}

func (b *Bot) refreshPresetList(ctx context.Context, tg *tgbot.Bot, chatID int64, msgID int, userID int64) {
	presets, err := b.store.GetPresets(userID)
	if err != nil {
		slog.Error("getPresets failed", "err", err)
		return
	}
	tg.EditMessageText(ctx, &tgbot.EditMessageTextParams{
		ChatID:      chatID,
		MessageID:   msgID,
		Text:        presetsText(presets),
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: presetsKeyboard(presets),
	})
}

func (b *Bot) handlePresetSkipCallback(ctx context.Context, tg *tgbot.Bot, update *models.Update) {
	defer tg.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})

	// ps:<assignment_id>:<task_id>:<day_of_week>
	parts := strings.SplitN(update.CallbackQuery.Data, ":", 4)
	if len(parts) != 4 {
		return
	}
	assignmentID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return
	}
	taskID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return
	}
	dayOfWeek, err := strconv.Atoi(parts[3])
	if err != nil || dayOfWeek < 0 || dayOfWeek > 6 {
		return
	}

	if err := b.store.SkipPresetTask(assignmentID, taskID); err != nil {
		slog.Error("skipPresetTask failed", "err", err)
		return
	}
	b.editDayView(ctx, tg, update, dayOfWeek)
}

func (b *Bot) setPendingPreset(telegramID int64, title string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.pendingPreset[telegramID] = title
}

func (b *Bot) getPendingPreset(telegramID int64) string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.pendingPreset[telegramID]
}

func (b *Bot) popPendingPreset(telegramID int64) string {
	b.mu.Lock()
	defer b.mu.Unlock()
	title := b.pendingPreset[telegramID]
	delete(b.pendingPreset, telegramID)
	return title
}

func (b *Bot) initPendingPresetDays(telegramID int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.pendingPresetDays[telegramID] = [7]bool{}
}

func (b *Bot) togglePendingPresetDay(telegramID int64, day int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	days := b.pendingPresetDays[telegramID]
	days[day] = !days[day]
	b.pendingPresetDays[telegramID] = days
}

func (b *Bot) getPendingPresetDays(telegramID int64) [7]bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.pendingPresetDays[telegramID]
}

func (b *Bot) popPendingPresetDays(telegramID int64) [7]bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	days := b.pendingPresetDays[telegramID]
	delete(b.pendingPresetDays, telegramID)
	return days
}

func (b *Bot) presetDayPrompt(ctx context.Context, tg *tgbot.Bot, chatID int64, title string) {
	tg.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: chatID,
		Text: fmt.Sprintf("Which days should <b>%s</b> repeat?\nTap days to select, then Save.",
			html.EscapeString(title)),
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: presetDayMultiKeyboard([7]bool{}),
	})
}
