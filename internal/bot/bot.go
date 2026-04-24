package bot

import (
	"context"
	"log/slog"
	"sync"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/mishkamashka/weekly-planner/internal/scheduler"
	"github.com/mishkamashka/weekly-planner/internal/store"
)

type userState int

const (
	stateIdle userState = iota
	stateWaitingForTask
	stateSettingTimezone
	stateSettingMorningTime
	stateSettingSundayTime
)

type Bot struct {
	tg      *tgbot.Bot
	ownerID int64
	store   *store.Store
	sched   *scheduler.Scheduler
	mu      sync.Mutex
	states  map[int64]userState
}

func New(token string, ownerID int64, store *store.Store) (*Bot, error) {
	b := &Bot{
		ownerID: ownerID,
		store:   store,
		states:  make(map[int64]userState),
	}

	tg, err := tgbot.New(token,
		tgbot.WithMiddlewares(b.ownerOnly()),
		tgbot.WithDefaultHandler(b.handleDefault),
	)
	if err != nil {
		return nil, err
	}
	b.tg = tg

	tg.RegisterHandler(tgbot.HandlerTypeMessageText, "/ping", tgbot.MatchTypeExact, b.handlePing)
	tg.RegisterHandler(tgbot.HandlerTypeMessageText, "/start", tgbot.MatchTypeExact, b.handleStart)
	tg.RegisterHandler(tgbot.HandlerTypeMessageText, "➕ Add task", tgbot.MatchTypeExact, b.handleAddTaskPrompt)
	tg.RegisterHandler(tgbot.HandlerTypeMessageText, "/add", tgbot.MatchTypePrefix, b.handleAddCommand)
	tg.RegisterHandler(tgbot.HandlerTypeMessageText, "/backlog", tgbot.MatchTypeExact, b.handleBacklog)
	tg.RegisterHandler(tgbot.HandlerTypeMessageText, "📋 Backlog", tgbot.MatchTypeExact, b.handleBacklog)
	tg.RegisterHandler(tgbot.HandlerTypeMessageText, "📌 Today", tgbot.MatchTypeExact, b.handleToday)
	tg.RegisterHandler(tgbot.HandlerTypeMessageText, "📅 Week", tgbot.MatchTypeExact, b.handleWeek)
	tg.RegisterHandler(tgbot.HandlerTypeMessageText, "⚙️ Settings", tgbot.MatchTypeExact, b.handleSettings)
	tg.RegisterHandler(tgbot.HandlerTypeMessageText, "← Back", tgbot.MatchTypeExact, b.handleBack)
	for _, day := range []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"} {
		tg.RegisterHandler(tgbot.HandlerTypeMessageText, day, tgbot.MatchTypeExact, b.handleDayButton)
	}
	tg.RegisterHandler(tgbot.HandlerTypeCallbackQueryData, "t:", tgbot.MatchTypePrefix, b.handleTaskCallback)
	tg.RegisterHandler(tgbot.HandlerTypeCallbackQueryData, "ck:", tgbot.MatchTypePrefix, b.handleCompleteCallback)
	tg.RegisterHandler(tgbot.HandlerTypeCallbackQueryData, "bl:", tgbot.MatchTypePrefix, b.handleBacklogCallback)
	tg.RegisterHandler(tgbot.HandlerTypeCallbackQueryData, "set:", tgbot.MatchTypePrefix, b.handleSettingsCallback)

	return b, nil
}

func (b *Bot) SetScheduler(s *scheduler.Scheduler) {
	b.sched = s
}

func (b *Bot) SendToUser(ctx context.Context, telegramID int64, text string) {
	b.tg.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: telegramID,
		Text:   text,
	})
}

func (b *Bot) SendDayView(ctx context.Context, telegramID int64, dayOfWeek int) {
	user, err := b.store.GetOrCreateUser(telegramID, "")
	if err != nil {
		slog.Error("getOrCreateUser failed in SendDayView", "err", err)
		return
	}
	tasks, err := b.store.GetDayTasks(user.ID, currentWeekMonday(), dayOfWeek)
	if err != nil {
		slog.Error("getDayTasks failed in SendDayView", "err", err)
		return
	}
	b.tg.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:      telegramID,
		Text:        dayTasksText(dayFullName[dayOfWeek], tasks),
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: dayTasksKeyboard(tasks, dayOfWeek),
	})
}

func (b *Bot) Run(ctx context.Context) error {
	slog.Info("telegram bot starting")
	b.tg.Start(ctx)
	return nil
}

func (b *Bot) ownerOnly() tgbot.Middleware {
	return func(next tgbot.HandlerFunc) tgbot.HandlerFunc {
		return func(ctx context.Context, tg *tgbot.Bot, update *models.Update) {
			if !b.isOwner(update) {
				return
			}
			next(ctx, tg, update)
		}
	}
}

func (b *Bot) isOwner(update *models.Update) bool {
	if update.Message != nil && update.Message.From != nil {
		return update.Message.From.ID == b.ownerID
	}
	if update.CallbackQuery != nil {
		return update.CallbackQuery.From.ID == b.ownerID
	}
	return false
}

func (b *Bot) setState(telegramID int64, s userState) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.states[telegramID] = s
}

func (b *Bot) getState(telegramID int64) userState {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.states[telegramID]
}
