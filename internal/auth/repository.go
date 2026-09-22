package auth

import (
	"context"
	"database/sql"
	"time"

	"github.com/musiermoore/oksana-vpn-api/internal/database/orm"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindByTelegram(
	ctx context.Context,
	telegram string,
) (User, error) {
	var user User

	err := orm.Query(r.db, ctx).
		Table("users").
		Select("id", "name", "telegram", "password").
		Where("telegram", "=", "@"+telegram).
		First(&user)

	return user, err
}

func (r *Repository) CreateToken(
	ctx context.Context,
	userID int64,
	hash [32]byte,
	expiresAt time.Time,
) error {
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO auth_tokens
         (token_hash, user_id, expires_at, created_at)
         VALUES (?, ?, ?, ?)`,
		hash[:],
		userID,
		expiresAt.UTC(),
		time.Now().UTC(),
	)

	return err
}

func (r *Repository) FindByToken(
	ctx context.Context,
	hash [32]byte,
) (User, error) {
	var user User

	err := r.db.QueryRowContext(
		ctx,
		`SELECT u.id, u.name, u.telegram
         FROM auth_tokens t
         JOIN users u ON u.id = t.user_id
         WHERE t.token_hash = ?
           AND t.expires_at > UTC_TIMESTAMP(6)`,
		hash[:],
	).Scan(
		&user.ID,
		&user.Name,
		&user.Telegram,
	)

	return user, err
}

func (r *Repository) DeleteToken(
	ctx context.Context,
	hash [32]byte,
) error {
	_, err := r.db.ExecContext(
		ctx,
		`DELETE FROM auth_tokens
         WHERE token_hash = ?`,
		hash[:],
	)

	return err
}
