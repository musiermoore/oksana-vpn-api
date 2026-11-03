-- +goose Up
ALTER TABLE telegram_app_tokens
    ADD COLUMN source ENUM('telegram', 'web') NOT NULL DEFAULT 'telegram' AFTER user_id;

-- +goose Down
ALTER TABLE telegram_app_tokens
    DROP COLUMN source;
