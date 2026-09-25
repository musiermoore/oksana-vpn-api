package types

import "net/url"

type TelegramUser struct {
	ID              int64  `json:"id"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	Username        string `json:"username"`
	LanguageCode    string `json:"language_code"`
	IsPremium       bool   `json:"is_premium"`
	AllowsWriteToPM bool   `json:"allows_write_to_pm"`
}

type TelegramInitData struct {
	QueryId   string       `json:"query_id"`
	User      TelegramUser `json:"user"`
	AuthDate  int64        `json:"auth_date"`
	Signature string       `json:"signature"`
	Hash      string       `json:"hash"`

	Raw url.Values
}

type TelegramAuthRequest struct {
	Data string `json:"data" binding:"required"`
}
