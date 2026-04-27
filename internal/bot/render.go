package bot

import (
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/go-telegram/bot/models"
	"github.com/mishkamashka/weekly-planner/internal/store"
)

func mainKeyboard() *models.ReplyKeyboardMarkup {
	return &models.ReplyKeyboardMarkup{
		Keyboard: [][]models.KeyboardButton{
			{{Text: "➕ Add task"}, {Text: "📌 Today"}},
			{{Text: "📋 Backlog"}, {Text: "📅 Week"}},
			{{Text: "📆 Plan week"}, {Text: "⚙️ Settings"}},
		},
		ResizeKeyboard: true,
	}
}

func settingsKeyboard() *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "🌍 Change timezone", CallbackData: "set:tz"}},
			{{Text: "⏰ Change morning time", CallbackData: "set:morning"}},
			{{Text: "📅 Change Sunday ping time", CallbackData: "set:sunday"}},
		},
	}
}

func weekReplyKeyboard() *models.ReplyKeyboardMarkup {
	return &models.ReplyKeyboardMarkup{
		Keyboard: [][]models.KeyboardButton{
			{{Text: "Mon"}, {Text: "Tue"}, {Text: "Wed"}, {Text: "Thu"}},
			{{Text: "Fri"}, {Text: "Sat"}, {Text: "Sun"}},
			{{Text: "← Back"}},
		},
		ResizeKeyboard: true,
	}
}

func taskKeyboard(taskID int64) *models.InlineKeyboardMarkup {
	id := fmt.Sprintf("%d", taskID)
	cb := func(day string) string { return "t:" + id + ":" + day }

	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "Mon", CallbackData: cb("0")},
				{Text: "Tue", CallbackData: cb("1")},
				{Text: "Wed", CallbackData: cb("2")},
				{Text: "Thu", CallbackData: cb("3")},
				{Text: "Fri", CallbackData: cb("4")},
			},
			{
				{Text: "Sat", CallbackData: cb("5")},
				{Text: "Sun", CallbackData: cb("6")},
				{Text: "🗄 Archive", CallbackData: cb("a")},
			},
		},
	}
}

func dayTasksText(dayName string, tasks []*store.AssignedTask) string {
	if len(tasks) == 0 {
		return "<b>" + dayName + "</b>\n\nNothing planned."
	}
	lines := []string{"<b>" + dayName + "</b>", ""}
	for _, t := range tasks {
		title := html.EscapeString(t.Title)
		if t.Completed {
			lines = append(lines, "• <s>"+title+"</s>")
		} else {
			lines = append(lines, "• "+title)
		}
	}
	return strings.Join(lines, "\n")
}

func dayTasksKeyboard(tasks []*store.AssignedTask, dayOfWeek int) *models.InlineKeyboardMarkup {
	day := strconv.Itoa(dayOfWeek)
	rows := [][]models.InlineKeyboardButton{}
	for _, t := range tasks {
		if t.Completed {
			continue
		}
		aid := strconv.FormatInt(t.AssignmentID, 10)
		tid := strconv.FormatInt(t.TaskID, 10)
		rows = append(rows, []models.InlineKeyboardButton{
			{Text: "✅ " + truncate(t.Title, 25), CallbackData: "ck:" + aid + ":" + day},
			{Text: "↩ backlog", CallbackData: "bl:" + aid + ":" + tid + ":" + day},
		})
	}
	return &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func planNextKeyboard(taskID int64) *models.InlineKeyboardMarkup {
	id := fmt.Sprintf("%d", taskID)
	cb := func(action string) string { return "pn:" + id + ":" + action }

	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "Mon", CallbackData: cb("0")},
				{Text: "Tue", CallbackData: cb("1")},
				{Text: "Wed", CallbackData: cb("2")},
				{Text: "Thu", CallbackData: cb("3")},
				{Text: "Fri", CallbackData: cb("4")},
			},
			{
				{Text: "Sat", CallbackData: cb("5")},
				{Text: "Sun", CallbackData: cb("6")},
				{Text: "⏭ Skip", CallbackData: cb("skip")},
				{Text: "🗄 Archive", CallbackData: cb("a")},
			},
		},
	}
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}
