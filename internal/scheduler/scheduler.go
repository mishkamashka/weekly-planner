package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mishkamashka/weekly-planner/internal/store"
	"github.com/robfig/cron/v3"
)

type SendTextFunc func(ctx context.Context, telegramID int64, text string)
type SendDayViewFunc func(ctx context.Context, telegramID int64, dayOfWeek int)

type Scheduler struct {
	store       *store.Store
	cron        *cron.Cron
	sendText    SendTextFunc
	sendDayView SendDayViewFunc
	mu          sync.Mutex
	entries     map[int64][]cron.EntryID // store userID → entry IDs
}

func New(store *store.Store, sendText SendTextFunc, sendDayView SendDayViewFunc) *Scheduler {
	return &Scheduler{
		store:       store,
		cron:        cron.New(),
		sendText:    sendText,
		sendDayView: sendDayView,
		entries:     make(map[int64][]cron.EntryID),
	}
}

func (s *Scheduler) Run(ctx context.Context) error {
	users, err := s.store.GetAllUsers()
	if err != nil {
		return fmt.Errorf("load users: %w", err)
	}

	for _, u := range users {
		if err := s.registerUser(u); err != nil {
			slog.Warn("failed to register cron for user", "userID", u.ID, "err", err)
		}
	}

	s.cron.Start()
	slog.Info("scheduler started", "users", len(users))

	<-ctx.Done()
	s.cron.Stop()
	return nil
}

func (s *Scheduler) ReloadUser(userID int64) {
	user, err := s.store.GetUserByID(userID)
	if err != nil {
		slog.Error("getUser failed on reload", "userID", userID, "err", err)
		return
	}

	s.mu.Lock()
	for _, id := range s.entries[userID] {
		s.cron.Remove(id)
	}
	delete(s.entries, userID)
	s.mu.Unlock()

	if err := s.registerUser(user); err != nil {
		slog.Error("registerUser failed on reload", "userID", userID, "err", err)
	}
}

func (s *Scheduler) registerUser(u *store.User) error {
	morningSpec, err := cronSpec(u.Timezone, u.MorningTime, "* * *")
	if err != nil {
		return fmt.Errorf("morning spec: %w", err)
	}
	sundaySpec, err := cronSpec(u.Timezone, u.SundayTime, "* * 0")
	if err != nil {
		return fmt.Errorf("sunday spec: %w", err)
	}

	telegramID := u.TelegramID

	morningID, err := s.cron.AddFunc(morningSpec, func() {
		s.sendDayView(context.Background(), telegramID, todayDayOfWeek())
	})
	if err != nil {
		return fmt.Errorf("add morning job: %w", err)
	}

	sundayID, err := s.cron.AddFunc(sundaySpec, func() {
		s.sendText(context.Background(), telegramID,
			"📋 Time to plan your week! Open 📅 Week to assign tasks to days.")
	})
	if err != nil {
		s.cron.Remove(morningID)
		return fmt.Errorf("add sunday job: %w", err)
	}

	s.mu.Lock()
	s.entries[u.ID] = append(s.entries[u.ID], morningID, sundayID)
	s.mu.Unlock()

	return nil
}

func cronSpec(timezone, timeStr, rest string) (string, error) {
	parts := strings.SplitN(timeStr, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid time: %s", timeStr)
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return "", fmt.Errorf("invalid hour in %s", timeStr)
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil || minute < 0 || minute > 59 {
		return "", fmt.Errorf("invalid minute in %s", timeStr)
	}
	return fmt.Sprintf("CRON_TZ=%s %d %d %s", timezone, minute, hour, rest), nil
}

func todayDayOfWeek() int {
	w := int(time.Now().Weekday())
	if w == 0 {
		return 6
	}
	return w - 1
}
