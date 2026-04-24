package bot

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"

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

func (b *Bot) handleBacklog(ctx context.Context, tg *tgbot.Bot, update *models.Update) {
	user, err := b.store.GetOrCreateUser(update.Message.From.ID, "")
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
			ChatID:      update.Message.Chat.ID,
			Text:        "Your backlog is empty. Tap ➕ Add task to add something.",
			ReplyMarkup: mainKeyboard(),
		})
		return
	}

	for _, task := range tasks {
		tg.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID:      update.Message.Chat.ID,
			Text:        task.Title,
			ReplyMarkup: taskKeyboard(task.ID),
		})
	}
}

func (b *Bot) handleTaskCallback(ctx context.Context, tg *tgbot.Bot, update *models.Update) {
	defer tg.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})

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

	if action == "a" {
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
		if err := b.store.AssignTask(taskID, user.ID, currentWeekMonday(), dayOfWeek); err != nil {
			slog.Error("assignTask failed", "err", err)
			return
		}
		days := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
		label = days[dayOfWeek]
	}

	_, err = tg.EditMessageText(ctx, &tgbot.EditMessageTextParams{
		ChatID:      msg.Chat.ID,
		MessageID:   msg.ID,
		Text:        msg.Text + " → " + label,
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{}},
	})
	if err != nil {
		slog.Error("editMessageText failed", "err", err)
	}
}

var dayIndex = map[string]int{
	"Mon": 0, "Tue": 1, "Wed": 2, "Thu": 3, "Fri": 4, "Sat": 5, "Sun": 6,
}

var dayFullName = []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}

func (b *Bot) handleWeek(ctx context.Context, tg *tgbot.Bot, update *models.Update) {
	tg.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        "📅",
		ReplyMarkup: weekReplyKeyboard(),
	})
}

func (b *Bot) handleBack(ctx context.Context, tg *tgbot.Bot, update *models.Update) {
	tg.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        "🏠",
		ReplyMarkup: mainKeyboard(),
	})
}

func (b *Bot) handleDayButton(ctx context.Context, tg *tgbot.Bot, update *models.Update) {
	dayOfWeek, ok := dayIndex[update.Message.Text]
	if !ok {
		return
	}

	user, err := b.store.GetOrCreateUser(update.Message.From.ID, "")
	if err != nil {
		slog.Error("getOrCreateUser failed", "err", err)
		return
	}

	tasks, err := b.store.GetDayTasks(user.ID, currentWeekMonday(), dayOfWeek)
	if err != nil {
		slog.Error("getDayTasks failed", "err", err)
		return
	}

	var text string
	if len(tasks) == 0 {
		text = dayFullName[dayOfWeek] + ": nothing planned."
	} else {
		lines := make([]string, len(tasks))
		for i, t := range tasks {
			lines[i] = "• " + t.Title
		}
		text = dayFullName[dayOfWeek] + ":\n" + strings.Join(lines, "\n")
	}

	tg.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   text,
	})
}

func currentWeekMonday() time.Time {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return now.AddDate(0, 0, -(weekday - 1)).Truncate(24 * time.Hour)
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
