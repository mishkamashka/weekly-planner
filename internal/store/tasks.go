package store

import "time"

type Task struct {
	ID        int64
	UserID    int64
	Title     string
	Status    string
	CreatedAt time.Time
}

func (s *Store) AddTask(userID int64, title string) (*Task, error) {
	res, err := s.db.Exec(
		`INSERT INTO tasks (user_id, title, status) VALUES (?, ?, 'backlog')`,
		userID, title,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &Task{ID: id, UserID: userID, Title: title, Status: "backlog"}, nil
}

func (s *Store) GetBacklog(userID int64) ([]*Task, error) {
	rows, err := s.db.Query(
		`SELECT id, user_id, title, status, created_at FROM tasks
		 WHERE user_id = ? AND status = 'backlog' ORDER BY created_at ASC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		t := &Task{}
		if err := rows.Scan(&t.ID, &t.UserID, &t.Title, &t.Status, &t.CreatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}
