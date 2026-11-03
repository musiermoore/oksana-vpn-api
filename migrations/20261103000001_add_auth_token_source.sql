-- +goose Up
ALTER TABLE auth_tokens
    ADD COLUMN source ENUM('telegram', 'web') NOT NULL DEFAULT 'telegram' AFTER user_id;

-- +goose Down
ALTER TABLE auth_tokens
    DROP COLUMN source;
