package bot

import (
	"context"
	"log/slog"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type Bot struct {
	tg      *tgbot.Bot
	ownerID int64
}

func New(token string, ownerID int64) (*Bot, error) {
	b := &Bot{ownerID: ownerID}

	tg, err := tgbot.New(token, tgbot.WithMiddlewares(b.ownerOnly()))
	if err != nil {
		return nil, err
	}
	b.tg = tg

	tg.RegisterHandler(tgbot.HandlerTypeMessageText, "/ping", tgbot.MatchTypeExact, b.handlePing)
	tg.RegisterHandler(tgbot.HandlerTypeMessageText, "/start", tgbot.MatchTypeExact, b.handleStart)

	return b, nil
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

func (b *Bot) handlePing(ctx context.Context, tg *tgbot.Bot, update *models.Update) {
	tg.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "pong",
	})
}

func (b *Bot) handleStart(ctx context.Context, tg *tgbot.Bot, update *models.Update) {
	tg.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Hey! I'm your weekly planner bot. Send /ping to check I'm alive.",
	})
}
