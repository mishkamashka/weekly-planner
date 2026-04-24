package store

import "time"

func (s *Store) AssignTask(taskID, userID int64, weekStart time.Time, dayOfWeek int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`INSERT INTO assignments (task_id, user_id, week_start, day_of_week) VALUES (?, ?, ?, ?)`,
		taskID, userID, weekStart.Format("2006-01-02"), dayOfWeek,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`UPDATE tasks SET status = 'assigned' WHERE id = ?`, taskID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) ArchiveTask(taskID int64) error {
	_, err := s.db.Exec(`UPDATE tasks SET status = 'archived' WHERE id = ?`, taskID)
	return err
}
