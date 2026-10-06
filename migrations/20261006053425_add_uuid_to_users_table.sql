-- +goose Up
ALTER TABLE users
    ADD COLUMN uuid CHAR(36) NULL AFTER id,
    ADD UNIQUE INDEX users_uuid_unique (uuid);

-- +goose Down
ALTER TABLE users
    DROP INDEX users_uuid_unique,
    DROP COLUMN uuid;