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
			{{Text: "📆 Plan week"}, {Text: "🔁 Presets"}},
			{{Text: "⚙️ Settings"}},
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
		var secondBtn models.InlineKeyboardButton
		if t.IsPreset {
			secondBtn = models.InlineKeyboardButton{Text: "⏭ skip", CallbackData: "ps:" + aid + ":" + tid + ":" + day}
		} else {
			secondBtn = models.InlineKeyboardButton{Text: "↩ backlog", CallbackData: "bl:" + aid + ":" + tid + ":" + day}
		}
		rows = append(rows, []models.InlineKeyboardButton{
			{Text: "✅ " + truncate(t.Title, 25), CallbackData: "ck:" + aid + ":" + day},
			secondBtn,
		})
	}
	return &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

type presetGroup struct {
	title   string
	days    []int
	active  bool
	firstID int64
}

func groupPresets(presets []*store.Preset) []presetGroup {
	// presets are ordered by title, day_of_week — so same-title rows are adjacent
	var groups []presetGroup
	for _, p := range presets {
		if len(groups) > 0 && groups[len(groups)-1].title == p.Title {
			groups[len(groups)-1].days = append(groups[len(groups)-1].days, p.DayOfWeek)
		} else {
			groups = append(groups, presetGroup{
				title:   p.Title,
				days:    []int{p.DayOfWeek},
				active:  p.Active,
				firstID: p.ID,
			})
		}
	}
	return groups
}

func presetsText(presets []*store.Preset) string {
	if len(presets) == 0 {
		return "🔁 <b>Recurring presets</b>\n\nNo presets yet. Tap ➕ to add one.\n\nActive presets are applied automatically when you run /plan."
	}
	return "🔁 <b>Recurring presets</b>\n\nActive presets are applied automatically when you run /plan.\nTap a preset to toggle it on/off."
}

func presetsKeyboard(presets []*store.Preset) *models.InlineKeyboardMarkup {
	dayNames := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	rows := [][]models.InlineKeyboardButton{}
	for _, g := range groupPresets(presets) {
		icon := "✅"
		if !g.active {
			icon = "⬜"
		}
		dayParts := make([]string, len(g.days))
		for i, d := range g.days {
			dayParts[i] = dayNames[d]
		}
		label := fmt.Sprintf("%s %s (%s)", icon, truncate(g.title, 18), strings.Join(dayParts, ", "))
		id := strconv.FormatInt(g.firstID, 10)
		rows = append(rows, []models.InlineKeyboardButton{
			{Text: label, CallbackData: "pr:toggle:" + id},
			{Text: "🗑", CallbackData: "pr:del:" + id},
		})
	}
	rows = append(rows, []models.InlineKeyboardButton{
		{Text: "➕ Add preset", CallbackData: "pr:new"},
	})
	return &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func presetDayMultiKeyboard(selected [7]bool) *models.InlineKeyboardMarkup {
	labels := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	row1 := make([]models.InlineKeyboardButton, 5)
	for i := 0; i < 5; i++ {
		label := labels[i]
		if selected[i] {
			label = "✅ " + label
		}
		row1[i] = models.InlineKeyboardButton{Text: label, CallbackData: "pr:day:" + strconv.Itoa(i)}
	}
	row2 := make([]models.InlineKeyboardButton, 2)
	for i := 5; i < 7; i++ {
		label := labels[i]
		if selected[i] {
			label = "✅ " + label
		}
		row2[i-5] = models.InlineKeyboardButton{Text: label, CallbackData: "pr:day:" + strconv.Itoa(i)}
	}
	count := 0
	for _, s := range selected {
		if s {
			count++
		}
	}
	saveLabel := "Save"
	if count > 0 {
		saveLabel = fmt.Sprintf("✅ Save (%d)", count)
	}
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			row1,
			row2,
			{{Text: saveLabel, CallbackData: "pr:day:done"}},
		},
	}
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
