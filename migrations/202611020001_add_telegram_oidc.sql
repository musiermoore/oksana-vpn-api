-- +goose Up
CREATE TABLE telegram_oidc_transactions (
    state_hash CHAR(64) NOT NULL,
    nonce VARCHAR(128) NOT NULL,
    pkce_verifier VARCHAR(128) NOT NULL,
    return_url VARCHAR(2048) NOT NULL,
    expires_at DATETIME(6) NOT NULL,
    created_at DATETIME(6) NOT NULL,
    PRIMARY KEY (state_hash),
    INDEX idx_telegram_oidc_transactions_expires_at (expires_at)
) ENGINE=InnoDB;

CREATE TABLE telegram_oidc_identities (
    subject VARCHAR(255) NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    created_at DATETIME(6) NOT NULL,
    PRIMARY KEY (subject),
    UNIQUE KEY uq_telegram_oidc_identities_user_id (user_id),
    CONSTRAINT fk_telegram_oidc_identities_user FOREIGN KEY (user_id)
        REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE telegram_oidc_login_tickets (
    ticket_hash CHAR(64) NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    expires_at DATETIME(6) NOT NULL,
    created_at DATETIME(6) NOT NULL,
    PRIMARY KEY (ticket_hash),
    INDEX idx_telegram_oidc_login_tickets_expires_at (expires_at),
    CONSTRAINT fk_telegram_oidc_login_tickets_user FOREIGN KEY (user_id)
        REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB;

-- +goose Down
DROP TABLE telegram_oidc_login_tickets;
DROP TABLE telegram_oidc_identities;
DROP TABLE telegram_oidc_transactions;
