package store

import "time"

type User struct {
	ID          int64
	TelegramID  int64
	Name        string
	Timezone    string
	MorningTime string
	SundayTime  string
	CreatedAt   time.Time
}

func (s *Store) GetOrCreateUser(telegramID int64, name string) (*User, error) {
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO users (telegram_id, name) VALUES (?, ?)`,
		telegramID, name,
	)
	if err != nil {
		return nil, err
	}

	var u User
	err = s.db.QueryRow(
		`SELECT id, telegram_id, name, timezone, morning_time, sunday_time, created_at
		 FROM users WHERE telegram_id = ?`,
		telegramID,
	).Scan(&u.ID, &u.TelegramID, &u.Name, &u.Timezone, &u.MorningTime, &u.SundayTime, &u.CreatedAt)
	return &u, err
}
