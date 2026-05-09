package store

import "time"

type Preset struct {
	ID        int64
	UserID    int64
	Title     string
	DayOfWeek int
	Active    bool
	CreatedAt time.Time
}

func (s *Store) AddPreset(userID int64, title string, dayOfWeek int) (*Preset, error) {
	res, err := s.db.Exec(
		`INSERT INTO presets (user_id, title, day_of_week) VALUES (?, ?, ?)`,
		userID, title, dayOfWeek,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &Preset{ID: id, UserID: userID, Title: title, DayOfWeek: dayOfWeek, Active: true}, nil
}

func (s *Store) GetPresets(userID int64) ([]*Preset, error) {
	rows, err := s.db.Query(
		`SELECT id, user_id, title, day_of_week, active, created_at
		 FROM presets WHERE user_id = ? ORDER BY title, day_of_week`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var presets []*Preset
	for rows.Next() {
		p := &Preset{}
		if err := rows.Scan(&p.ID, &p.UserID, &p.Title, &p.DayOfWeek, &p.Active, &p.CreatedAt); err != nil {
			return nil, err
		}
		presets = append(presets, p)
	}
	return presets, rows.Err()
}

func (s *Store) TogglePreset(presetID int64) error {
	var userID int64
	var title string
	var active bool
	if err := s.db.QueryRow(
		`SELECT user_id, title, active FROM presets WHERE id = ?`, presetID,
	).Scan(&userID, &title, &active); err != nil {
		return err
	}
	_, err := s.db.Exec(
		`UPDATE presets SET active = ? WHERE user_id = ? AND title = ?`,
		!active, userID, title,
	)
	return err
}

func (s *Store) DeletePreset(presetID int64) error {
	var userID int64
	var title string
	if err := s.db.QueryRow(
		`SELECT user_id, title FROM presets WHERE id = ?`, presetID,
	).Scan(&userID, &title); err != nil {
		return err
	}
	_, err := s.db.Exec(`DELETE FROM presets WHERE user_id = ? AND title = ?`, userID, title)
	return err
}

// ApplyPresetsForWeek creates a task+assignment for each active preset not yet applied to weekStart.
// Returns the number of newly created assignments.
func (s *Store) ApplyPresetsForWeek(userID int64, weekStart time.Time) (int, error) {
	presets, err := s.GetPresets(userID)
	if err != nil {
		return 0, err
	}
	weekStr := weekStart.Format("2006-01-02")
	count := 0
	for _, p := range presets {
		if !p.Active {
			continue
		}
		var existing int
		if err := s.db.QueryRow(
			`SELECT COUNT(*) FROM assignments WHERE user_id = ? AND week_start = ? AND preset_id = ?`,
			userID, weekStr, p.ID,
		).Scan(&existing); err != nil || existing > 0 {
			continue
		}
		tx, err := s.db.Begin()
		if err != nil {
			return count, err
		}
		res, err := tx.Exec(
			`INSERT INTO tasks (user_id, title, status) VALUES (?, ?, 'assigned')`,
			userID, p.Title,
		)
		if err != nil {
			tx.Rollback()
			return count, err
		}
		taskID, _ := res.LastInsertId()
		_, err = tx.Exec(
			`INSERT INTO assignments (task_id, user_id, week_start, day_of_week, preset_id) VALUES (?, ?, ?, ?, ?)`,
			taskID, userID, weekStr, p.DayOfWeek, p.ID,
		)
		if err != nil {
			tx.Rollback()
			return count, err
		}
		if err := tx.Commit(); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// SkipPresetTask removes a preset-generated assignment and its ephemeral task entirely.
func (s *Store) SkipPresetTask(assignmentID, taskID int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM assignments WHERE id = ?`, assignmentID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM tasks WHERE id = ?`, taskID); err != nil {
		return err
	}
	return tx.Commit()
}
