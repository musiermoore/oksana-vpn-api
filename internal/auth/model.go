package auth

type User struct {
	ID           int64  `json:"id" db:"id"`
	Name         string `json:"name" db:"name"`
	Telegram     string `json:"telegram" db:"telegram"`
	PasswordHash string `json:"-" db:"password"`
}
