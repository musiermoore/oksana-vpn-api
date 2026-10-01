package auth

import (
	"time"
)

const USERS_TABLE = "users"
const AUTH_TOKENS_TABLE = "telegram_app_tokens"

type User struct {
	ID           int64  `json:"id" db:"id"`
	Name         string `json:"name" db:"name"`
	Telegram     string `json:"telegram" db:"telegram"`
	PasswordHash string `json:"-" db:"password"`
}

type AuthToken struct {
	TokenHash  string    `db:"token_hash"`
	UserId     int64     `db:"user_id"`
	ExpiresAt  time.Time `db:"expires_at"`
	LastUsedAt time.Time `db:"last_used_at"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}
