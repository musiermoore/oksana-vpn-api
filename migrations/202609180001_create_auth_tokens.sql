-- +goose Up

CREATE TABLE auth_tokens (
    token_hash BINARY(32) NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,

    expires_at DATETIME(6) NOT NULL,
    created_at DATETIME(6) NOT NULL,

    PRIMARY KEY (token_hash),
    INDEX idx_auth_tokens_user_id (user_id),
    INDEX idx_auth_tokens_expires_at (expires_at),
    CONSTRAINT fk_auth_tokens_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
) ENGINE=InnoDB;

-- +goose Down

DROP TABLE auth_tokens;