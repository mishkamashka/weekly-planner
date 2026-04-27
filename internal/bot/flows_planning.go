package bot

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (b *Bot) handlePlan(ctx context.Context, tg *tgbot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID
	telegramID := update.Message.From.ID

	user, err := b.store.GetOrCreateUser(telegramID, "")
	if err != nil {
		slog.Error("getOrCreateUser failed", "err", err)
		return
	}

	tasks, err := b.store.GetBacklog(user.ID)
	if err != nil {
		slog.Error("getBacklog failed", "err", err)
		return
	}

	if len(tasks) == 0 {
		tg.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: chatID,
			Text:   "Your backlog is empty — nothing to plan! Add tasks first.",
		})
		return
	}

	weekOf := nextWeekMonday().Format("Jan 2")
	tg.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: chatID,
		Text:   fmt.Sprintf("📆 <b>Plan next week</b> (week of %s)\n\nAssign tasks to days, skip, or archive:", weekOf),
		ParseMode: models.ParseModeHTML,
	})

	for _, task := range tasks {
		tg.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID:      chatID,
			Text:        task.Title,
			ReplyMarkup: planNextKeyboard(task.ID),
		})
	}
}

func (b *Bot) handlePlanNextCallback(ctx context.Context, tg *tgbot.Bot, update *models.Update) {
	defer tg.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})

	// pn:<task_id>:<action>  action = 0-6 | archive
	parts := strings.Split(update.CallbackQuery.Data, ":")
	if len(parts) != 3 {
		return
	}
	taskID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return
	}
	action := parts[2]

	msg := update.CallbackQuery.Message.Message
	if msg == nil {
		return
	}

	var label string
	if action == "skip" {
		label = "skipped"
	} else if action == "a" {
		if err := b.store.ArchiveTask(taskID); err != nil {
			slog.Error("archiveTask failed", "err", err)
			return
		}
		label = "archived"
	} else {
		dayOfWeek, err := strconv.Atoi(action)
		if err != nil || dayOfWeek < 0 || dayOfWeek > 6 {
			return
		}
		user, err := b.store.GetOrCreateUser(update.CallbackQuery.From.ID, "")
		if err != nil {
			slog.Error("getOrCreateUser failed", "err", err)
			return
		}
		if err := b.store.AssignTask(taskID, user.ID, nextWeekMonday(), dayOfWeek); err != nil {
			slog.Error("assignTask failed", "err", err)
			return
		}
		days := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
		label = days[dayOfWeek]
	}

	tg.EditMessageText(ctx, &tgbot.EditMessageTextParams{
		ChatID:      msg.Chat.ID,
		MessageID:   msg.ID,
		Text:        msg.Text + " → " + label,
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{}},
	})
}

func nextWeekMonday() time.Time {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	daysUntilNextMonday := 8 - weekday
	return now.AddDate(0, 0, daysUntilNextMonday).Truncate(24 * time.Hour)
}
