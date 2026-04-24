package bot

import (
	"fmt"

	"github.com/go-telegram/bot/models"
)

func mainKeyboard() *models.ReplyKeyboardMarkup {
	return &models.ReplyKeyboardMarkup{
		Keyboard: [][]models.KeyboardButton{
			{{Text: "➕ Add task"}, {Text: "📋 Backlog"}, {Text: "📅 Week"}},
		},
		ResizeKeyboard: true,
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
