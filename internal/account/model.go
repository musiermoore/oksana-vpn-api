package account

import (
	"time"

	"github.com/musiermoore/oksana-vpn-api/internal/referral"
)

// Profile is the response returned by GET /api/v1/auth/me.
type Profile struct {
	ID                               int64              `json:"id"`
	UUID                             string             `json:"uuid"`
	Name                             string             `json:"name"`
	Telegram                         *string            `json:"telegram"`
	TelegramID                       *string            `json:"telegram_id"`
	Balance                          float64            `json:"balance"`
	Debt                             float64            `json:"debt"`
	IsAdmin                          bool               `json:"is_admin"`
	HasActiveAccess                  bool               `json:"has_active_access"`
	HasVlessWlConfigs                bool               `json:"has_vless_wl_configs"`
	SubscriptionExpiresAt            *time.Time         `json:"subscription_expires_at"`
	HasMoneyForNextSubscriptionMonth bool               `json:"has_money_for_next_subscription_month"`
	ReferrerID                       *int64             `json:"referrer_id"`
	JoinAt                           *time.Time         `json:"join_at"`
	Referral                         referral.Summary   `json:"referral"`
	SubscriptionCodes                []SubscriptionCode `json:"subscription_codes"`
}

type SubscriptionCode struct {
	ID                int        `json:"id"`
	Code              string     `json:"code"`
	Months            *int       `json:"months"`
	Days              *int       `json:"days"`
	Price             float64    `json:"price"`
	Status            string     `json:"status"`
	ActivatedAt       *time.Time `json:"activated_at"`
	ExpiresAt         *time.Time `json:"expires_at"`
	ActivatedByUserID *int       `json:"activated_by_user_id"`
	CreatedAt         *time.Time `json:"created_at"`
}
