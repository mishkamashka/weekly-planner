-- +goose Up
CREATE TABLE assignments (
    id           INTEGER PRIMARY KEY,
    task_id      INTEGER NOT NULL REFERENCES tasks(id),
    user_id      INTEGER NOT NULL REFERENCES users(id),
    week_start   DATE NOT NULL,
    day_of_week  INTEGER NOT NULL,
    completed    BOOLEAN NOT NULL DEFAULT 0,
    completed_at TIMESTAMP
);

-- +goose Down
DROP TABLE assignments;
