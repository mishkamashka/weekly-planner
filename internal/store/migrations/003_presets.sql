-- +goose Up
CREATE TABLE presets (
    id          INTEGER PRIMARY KEY,
    user_id     INTEGER NOT NULL REFERENCES users(id),
    title       TEXT NOT NULL,
    day_of_week INTEGER NOT NULL,
    active      BOOLEAN NOT NULL DEFAULT 1,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE assignments ADD COLUMN preset_id INTEGER REFERENCES presets(id);

-- +goose Down
DROP TABLE presets;
