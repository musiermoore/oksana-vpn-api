package auth

import (
    "database/sql"
    "time"

    "github.com/musiermoore/oksana-vpn-api/internal/services/referral"
)

const USERS_TABLE = "users"
const AUTH_TOKENS_TABLE = "telegram_app_tokens"

type SubscriptionCode struct {
    ID                int        `json:"id" db:"id"`
    Code              string     `json:"code" db:"code"`
    Months            int        `json:"months" db:"months"`
    Days              int        `json:"days" db:"days"`
    Price             float64    `json:"price" db:"price"`
    Status            string     `json:"status" db:"status"`
    ActivatedAt       *time.Time `json:"activated_at" db:"activated_at"`
    ExpiresAt         *time.Time `json:"expires_at" db:"expires_at"`
    ActivatedByUserID *int       `json:"activated_by_user_id" db:"activated_by_user_id"`
    CreatedAt         time.Time  `json:"created_at" db:"created_at"`
}

type User struct {
    ID                                 int64      `json:"id" db:"id"`
    UUID                               string     `json:"uuid" db:"uuid"`
    Name                               string     `json:"name" db:"name"`
    Telegram                           *string    `json:"telegram" db:"telegram"`
    TelegramID                         *string    `json:"telegram_id" db:"telegram_id"`
    Balance                            float64    `json:"balance" db:"balance"`
    Debt                               float64    `json:"debt" db:"debt"`
    IsAdmin                            bool       `json:"is_admin" db:"is_admin"`
    HasActiveAccess                    bool       `json:"has_active_access" db:"has_active_access"`
    HasVlessWlConfigs                  bool       `json:"has_vless_wl_configs" db:"has_vless_wl_configs"`
    SubscriptionExpiresAt              *time.Time `json:"subscription_expires_at" db:"subscription_expires_at"`
    HasMoneyForNextSubscriptionMonth   bool       `json:"has_money_for_next_subscription_month" db:"has_money_for_next_subscription_month"`
    ReferrerID                         *int64     `json:"referrer_id" db:"referrer_id"`
    JoinAt                             *time.Time `json:"join_at" db:"join_at"`
    ReferralAccumulatedDiscountPercent int        `json:"-" db:"referral_accumulated_discount_percent"`

    Referral          referral.Summary   `json:"referral" db:"-"`
    SubscriptionCodes []SubscriptionCode `json:"subscription_codes" db:"-"`

    PasswordHash sql.NullString `json:"-" db:"password"`
}

type AuthToken struct {
    TokenHash  string    `db:"token_hash"`
    UserId     int64     `db:"user_id"`
    ExpiresAt  time.Time `db:"expires_at"`
    LastUsedAt time.Time `db:"last_used_at"`
    CreatedAt  time.Time `db:"created_at"`
    UpdatedAt  time.Time `db:"updated_at"`
}
