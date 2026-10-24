package auth

import (
	"database/sql"
	"time"
)

const usersTable = "users"
const authTokensTable = "telegram_app_tokens"

// User contains only the fields needed to log in and attach a referral.
type User struct {
	ID                                 int64          `db:"id"`
	UUID                               *string        `db:"uuid"`
	Name                               string         `db:"name"`
	Telegram                           *string        `db:"telegram"`
	TelegramID                         *string        `db:"telegram_id"`
	ReferrerID                         *int64         `db:"referrer_id"`
	JoinAt                             *time.Time     `db:"join_at"`
	ReferralAccumulatedDiscountPercent int            `db:"referral_accumulated_discount_percent"`
	UpdatedAt                          *time.Time     `db:"updated_at"`
	PasswordHash                       sql.NullString `db:"password"`
}

type AuthToken struct {
	TokenHash  string    `db:"token_hash"`
	UserID     int64     `db:"user_id"`
	ExpiresAt  time.Time `db:"expires_at"`
	LastUsedAt time.Time `db:"last_used_at"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}
