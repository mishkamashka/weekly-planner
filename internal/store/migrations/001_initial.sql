-- +goose Up
CREATE TABLE users (
    id           INTEGER PRIMARY KEY,
    telegram_id  INTEGER UNIQUE NOT NULL,
    name         TEXT,
    timezone     TEXT NOT NULL DEFAULT 'Europe/Berlin',
    morning_time TEXT NOT NULL DEFAULT '08:00',
    sunday_time  TEXT NOT NULL DEFAULT '18:00',
    created_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE tasks (
    id         INTEGER PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id),
    title      TEXT NOT NULL,
    notes      TEXT,
    status     TEXT NOT NULL DEFAULT 'backlog',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    done_at    TIMESTAMP
);

-- +goose Down
DROP TABLE tasks;
DROP TABLE users;
