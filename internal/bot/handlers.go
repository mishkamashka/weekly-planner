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
	telegramID := update.Message.From.ID
	text := strings.TrimSpace(update.Message.Text)

	switch b.getState(telegramID) {
	case stateWaitingForTask:
		if text == "" {
			return
		}
		b.setState(telegramID, stateIdle)
		b.addTask(ctx, tg, update.Message.Chat.ID, telegramID, text)

	case stateSettingTimezone:
		b.setState(telegramID, stateIdle)
		b.saveTimezone(ctx, tg, update.Message.Chat.ID, telegramID, text)

	case stateSettingMorningTime:
		b.setState(telegramID, stateIdle)
		b.saveMorningTime(ctx, tg, update.Message.Chat.ID, telegramID, text)

	case stateSettingSundayTime:
		b.setState(telegramID, stateIdle)
		b.saveSundayTime(ctx, tg, update.Message.Chat.ID, telegramID, text)

	default:
		tg.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID:      update.Message.Chat.ID,
			Text:        "Tap ➕ Add task to add something to your backlog.",
			ReplyMarkup: mainKeyboard(),
		})
	}
}

func (b *Bot) handleSettings(ctx context.Context, tg *tgbot.Bot, update *models.Update) {
	user, err := b.store.GetOrCreateUser(update.Message.From.ID, "")
	if err != nil {
		slog.Error("getOrCreateUser failed", "err", err)
		return
	}
	tg.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text: fmt.Sprintf("⚙️ <b>Settings</b>\n\n🌍 Timezone: <code>%s</code>\n⏰ Morning reminder: <code>%s</code>\n📅 Sunday ping: <code>%s</code>",
			user.Timezone, user.MorningTime, user.SundayTime),
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: settingsKeyboard(),
	})
}

func (b *Bot) handleSettingsCallback(ctx context.Context, tg *tgbot.Bot, update *models.Update) {
	defer tg.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})

	telegramID := update.CallbackQuery.From.ID
	chatID := update.CallbackQuery.Message.Message.Chat.ID

	switch update.CallbackQuery.Data {
	case "set:tz":
		b.setState(telegramID, stateSettingTimezone)
		tg.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: chatID,
			Text:   "Send your timezone (e.g. <code>Europe/Berlin</code>, <code>America/New_York</code>).",
			ParseMode: models.ParseModeHTML,
		})
	case "set:morning":
		b.setState(telegramID, stateSettingMorningTime)
		tg.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: chatID,
			Text:   "Send morning reminder time in <code>HH:MM</code> format (e.g. <code>08:00</code>).",
			ParseMode: models.ParseModeHTML,
		})
	case "set:sunday":
		b.setState(telegramID, stateSettingSundayTime)
		tg.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: chatID,
			Text:   "Send Sunday ping time in <code>HH:MM</code> format (e.g. <code>18:00</code>).",
			ParseMode: models.ParseModeHTML,
		})
	}
}

func (b *Bot) saveTimezone(ctx context.Context, tg *tgbot.Bot, chatID, telegramID int64, timezone string) {
	if _, err := time.LoadLocation(timezone); err != nil {
		tg.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: chatID,
			Text:   "Invalid timezone. Try something like <code>Europe/Berlin</code> or <code>America/New_York</code>.",
			ParseMode: models.ParseModeHTML,
		})
		b.setState(telegramID, stateSettingTimezone)
		return
	}
	user, err := b.store.GetOrCreateUser(telegramID, "")
	if err != nil {
		slog.Error("getOrCreateUser failed", "err", err)
		return
	}
	if err := b.store.UpdateTimezone(user.ID, timezone); err != nil {
		slog.Error("updateTimezone failed", "err", err)
		return
	}
	if b.sched != nil {
		b.sched.ReloadUser(user.ID)
	}
	tg.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: chatID,
		Text:   "✅ Timezone updated to <code>" + timezone + "</code>.",
		ParseMode: models.ParseModeHTML,
	})
}

func (b *Bot) saveMorningTime(ctx context.Context, tg *tgbot.Bot, chatID, telegramID int64, t string) {
	if !isValidTime(t) {
		tg.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: chatID,
			Text:   "Invalid format. Send time as <code>HH:MM</code>, e.g. <code>08:00</code>.",
			ParseMode: models.ParseModeHTML,
		})
		b.setState(telegramID, stateSettingMorningTime)
		return
	}
	user, err := b.store.GetOrCreateUser(telegramID, "")
	if err != nil {
		slog.Error("getOrCreateUser failed", "err", err)
		return
	}
	if err := b.store.UpdateMorningTime(user.ID, t); err != nil {
		slog.Error("updateMorningTime failed", "err", err)
		return
	}
	if b.sched != nil {
		b.sched.ReloadUser(user.ID)
	}
	tg.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: chatID,
		Text:   "✅ Morning reminder set to <code>" + t + "</code>.",
		ParseMode: models.ParseModeHTML,
	})
}

func (b *Bot) saveSundayTime(ctx context.Context, tg *tgbot.Bot, chatID, telegramID int64, t string) {
	if !isValidTime(t) {
		tg.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: chatID,
			Text:   "Invalid format. Send time as <code>HH:MM</code>, e.g. <code>18:00</code>.",
			ParseMode: models.ParseModeHTML,
		})
		b.setState(telegramID, stateSettingSundayTime)
		return
	}
	user, err := b.store.GetOrCreateUser(telegramID, "")
	if err != nil {
		slog.Error("getOrCreateUser failed", "err", err)
		return
	}
	if err := b.store.UpdateSundayTime(user.ID, t); err != nil {
		slog.Error("updateSundayTime failed", "err", err)
		return
	}
	if b.sched != nil {
		b.sched.ReloadUser(user.ID)
	}
	tg.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: chatID,
		Text:   "✅ Sunday ping set to <code>" + t + "</code>.",
		ParseMode: models.ParseModeHTML,
	})
}

func isValidTime(t string) bool {
	parts := strings.SplitN(t, ":", 2)
	if len(parts) != 2 {
		return false
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	return err1 == nil && err2 == nil && h >= 0 && h <= 23 && m >= 0 && m <= 59
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

func (b *Bot) handleToday(ctx context.Context, tg *tgbot.Bot, update *models.Update) {
	b.sendDayView(ctx, tg, update.Message.Chat.ID, update.Message.From.ID, todayDayOfWeek())
}

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
	b.sendDayView(ctx, tg, update.Message.Chat.ID, update.Message.From.ID, dayOfWeek)
}

func (b *Bot) handleCompleteCallback(ctx context.Context, tg *tgbot.Bot, update *models.Update) {
	defer tg.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})

	// ck:<assignment_id>:<day_of_week>
	parts := strings.SplitN(update.CallbackQuery.Data, ":", 3)
	if len(parts) != 3 {
		return
	}
	assignmentID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return
	}
	dayOfWeek, err := strconv.Atoi(parts[2])
	if err != nil || dayOfWeek < 0 || dayOfWeek > 6 {
		return
	}

	if err := b.store.CompleteAssignment(assignmentID); err != nil {
		slog.Error("completeAssignment failed", "err", err)
		return
	}
	b.editDayView(ctx, tg, update, dayOfWeek)
}

func (b *Bot) handleBacklogCallback(ctx context.Context, tg *tgbot.Bot, update *models.Update) {
	defer tg.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})

	// bl:<assignment_id>:<task_id>:<day_of_week>
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

	if err := b.store.MoveToBacklog(assignmentID, taskID); err != nil {
		slog.Error("moveToBacklog failed", "err", err)
		return
	}
	b.editDayView(ctx, tg, update, dayOfWeek)
}

func (b *Bot) sendDayView(ctx context.Context, tg *tgbot.Bot, chatID, telegramID int64, dayOfWeek int) {
	user, err := b.store.GetOrCreateUser(telegramID, "")
	if err != nil {
		slog.Error("getOrCreateUser failed", "err", err)
		return
	}
	tasks, err := b.store.GetDayTasks(user.ID, currentWeekMonday(), dayOfWeek)
	if err != nil {
		slog.Error("getDayTasks failed", "err", err)
		return
	}
	tg.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:    chatID,
		Text:      dayTasksText(dayFullName[dayOfWeek], tasks),
		ParseMode: models.ParseModeHTML,
		ReplyMarkup: dayTasksKeyboard(tasks, dayOfWeek),
	})
}

func (b *Bot) editDayView(ctx context.Context, tg *tgbot.Bot, update *models.Update, dayOfWeek int) {
	msg := update.CallbackQuery.Message.Message
	if msg == nil {
		return
	}
	user, err := b.store.GetOrCreateUser(update.CallbackQuery.From.ID, "")
	if err != nil {
		slog.Error("getOrCreateUser failed", "err", err)
		return
	}
	tasks, err := b.store.GetDayTasks(user.ID, currentWeekMonday(), dayOfWeek)
	if err != nil {
		slog.Error("getDayTasks failed", "err", err)
		return
	}
	_, err = tg.EditMessageText(ctx, &tgbot.EditMessageTextParams{
		ChatID:      msg.Chat.ID,
		MessageID:   msg.ID,
		Text:        dayTasksText(dayFullName[dayOfWeek], tasks),
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: dayTasksKeyboard(tasks, dayOfWeek),
	})
	if err != nil {
		slog.Error("editMessageText failed", "err", err)
	}
}

func (b *Bot) addTask(ctx context.Context, tg *tgbot.Bot, chatID, telegramID int64, input string) {
	user, err := b.store.GetOrCreateUser(telegramID, "")
	if err != nil {
		slog.Error("getOrCreateUser failed", "err", err)
		return
	}

	var titles []string
	for _, line := range strings.Split(input, "\n") {
		if t := strings.TrimSpace(line); t != "" {
			titles = append(titles, t)
		}
	}

	for _, title := range titles {
		if _, err := b.store.AddTask(user.ID, title); err != nil {
			slog.Error("addTask failed", "err", err, "title", title)
		}
	}

	var text string
	if len(titles) == 1 {
		text = "Added to backlog: " + titles[0]
	} else {
		text = fmt.Sprintf("Added %d tasks to backlog.", len(titles))
	}

	tg.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ReplyMarkup: mainKeyboard(),
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

func todayDayOfWeek() int {
	w := int(time.Now().Weekday())
	if w == 0 {
		return 6
	}
	return w - 1
}
