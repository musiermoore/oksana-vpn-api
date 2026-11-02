-- +goose Up
ALTER TABLE users
    ADD COLUMN login VARCHAR(64) NULL AFTER telegram,
    ADD UNIQUE INDEX users_login_unique (login);

-- +goose Down
ALTER TABLE users
    DROP INDEX users_login_unique,
    DROP COLUMN login;
