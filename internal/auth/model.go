package auth

import (
	"time"
)

const USERS_TABLE = "users"
const AUTH_TOKENS_TABLE = "telegram_app_tokens"

type User struct {
	ID                               int64   `json:"id" db:"id"`
	Name                             string  `json:"name" db:"name"`
	Telegram                         *string `json:"telegram" db:"telegram"`
	TelegramID                       *string `json:"telegram_id" db:"telegram_id"`
	Balance                          float64 `json:"balance" db:"balance"`
	Debt                             float64 `json:"debt" db:"debt"`
	IsAdmin                          bool    `json:"is_admin" db:"is_admin"`
	HasActiveAccess                  bool    `json:"has_active_access" db:"has_active_access"`
	HasVlessWlConfigs                bool    `json:"has_vless_wl_configs" db:"has_vless_wl_configs"`
	SubscriptionExpiresAt            *string `json:"subscription_expires_at" db:"subscription_expires_at"`
	HasMoneyForNextSubscriptionMonth bool    `json:"has_money_for_next_subscription_month" db:"has_money_for_next_subscription_month"`

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
