-- +goose Up
ALTER TABLE user_tokens
    DROP COLUMN password;

-- +goose Down
ALTER TABLE user_tokens
    ADD COLUMN password VARCHAR(255) NOT NULL DEFAULT '';
