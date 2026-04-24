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

func (s *Store) GetUserByID(id int64) (*User, error) {
	var u User
	err := s.db.QueryRow(
		`SELECT id, telegram_id, name, timezone, morning_time, sunday_time, created_at
		 FROM users WHERE id = ?`,
		id,
	).Scan(&u.ID, &u.TelegramID, &u.Name, &u.Timezone, &u.MorningTime, &u.SundayTime, &u.CreatedAt)
	return &u, err
}

func (s *Store) GetAllUsers() ([]*User, error) {
	rows, err := s.db.Query(
		`SELECT id, telegram_id, name, timezone, morning_time, sunday_time, created_at FROM users`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		u := &User{}
		if err := rows.Scan(&u.ID, &u.TelegramID, &u.Name, &u.Timezone, &u.MorningTime, &u.SundayTime, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (s *Store) UpdateTimezone(userID int64, timezone string) error {
	_, err := s.db.Exec(`UPDATE users SET timezone = ? WHERE id = ?`, timezone, userID)
	return err
}

func (s *Store) UpdateMorningTime(userID int64, t string) error {
	_, err := s.db.Exec(`UPDATE users SET morning_time = ? WHERE id = ?`, t, userID)
	return err
}

func (s *Store) UpdateSundayTime(userID int64, t string) error {
	_, err := s.db.Exec(`UPDATE users SET sunday_time = ? WHERE id = ?`, t, userID)
	return err
}
