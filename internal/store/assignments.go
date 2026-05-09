package store

import (
	"errors"
	"time"
)

var ErrAlreadyAssigned = errors.New("task already assigned to this slot")

func (s *Store) AssignTask(taskID, userID int64, weekStart time.Time, dayOfWeek int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var existing int
	if err := tx.QueryRow(
		`SELECT COUNT(*) FROM assignments WHERE task_id = ? AND week_start = ? AND day_of_week = ? AND completed = 0`,
		taskID, weekStart.Format("2006-01-02"), dayOfWeek,
	).Scan(&existing); err != nil {
		return err
	}
	if existing > 0 {
		return ErrAlreadyAssigned
	}

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

func (s *Store) CompleteAssignment(assignmentID int64) error {
	_, err := s.db.Exec(
		`UPDATE assignments SET completed = 1, completed_at = CURRENT_TIMESTAMP WHERE id = ?`,
		assignmentID,
	)
	return err
}

// RestoreOverdueTasks moves uncompleted non-preset tasks from past days back to
// backlog status without removing their assignments (history is preserved).
func (s *Store) RestoreOverdueTasks(userID int64) error {
	var timezone string
	if err := s.db.QueryRow(`SELECT timezone FROM users WHERE id = ?`, userID).Scan(&timezone); err != nil {
		timezone = "UTC"
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}
	today := time.Now().In(loc).Format("2006-01-02")
	_, err = s.db.Exec(`
		UPDATE tasks SET status = 'backlog'
		WHERE status = 'assigned'
		AND id IN (
			SELECT task_id FROM assignments
			WHERE user_id = ?
			AND completed = 0
			AND preset_id IS NULL
			AND date(week_start, '+' || day_of_week || ' days') < ?
		)
		AND id NOT IN (
			SELECT task_id FROM assignments
			WHERE user_id = ?
			AND completed = 0
			AND date(week_start, '+' || day_of_week || ' days') >= ?
		)`, userID, today, userID, today)
	return err
}

func (s *Store) MoveToBacklog(assignmentID, taskID int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM assignments WHERE id = ?`, assignmentID); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE tasks SET status = 'backlog' WHERE id = ?`, taskID); err != nil {
		return err
	}
	return tx.Commit()
}
