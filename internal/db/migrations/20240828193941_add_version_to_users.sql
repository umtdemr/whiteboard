-- +goose Up
ALTER TABLE users ADD COLUMN version integer NOT NULL DEFAULT 1;

-- +goose Down
ALTER TABLE users DROP COLUMN version;
