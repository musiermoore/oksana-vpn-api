package auth

import (
	"time"
)

type User struct {
	ID           int64  `json:"id" db:"id"`
	Name         string `json:"name" db:"name"`
	Telegram     string `json:"telegram" db:"telegram"`
	PasswordHash string `json:"-" db:"password"`
}

type AuthToken struct {
	TokenHash []byte    `db:"token_hash"`
	UserId    int64     `db:"user_id"`
	ExpiresAt time.Time `db:"expires_at"`
	CreatedAt time.Time `db:"created_at"`
}
