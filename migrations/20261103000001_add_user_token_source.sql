-- +goose Up
ALTER TABLE user_tokens
    ADD COLUMN source ENUM('telegram', 'web') NOT NULL DEFAULT 'telegram' AFTER user_id;

-- +goose Down
ALTER TABLE user_tokens
    DROP COLUMN source;
