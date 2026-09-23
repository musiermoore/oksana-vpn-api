package auth

import (
	"context"
	"database/sql"
	"time"

	"github.com/musiermoore/oksana-vpn-api/internal/database/builder"
	"github.com/musiermoore/oksana-vpn-api/internal/database/builder/query/expression"
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

	err := builder.Query(r.db, ctx).
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
	return builder.Query(r.db, ctx).
		Table("auth_tokens").
		Insert(AuthToken{
			TokenHash: hash[:],
			UserId:    userID,
			ExpiresAt: expiresAt.UTC(),
			CreatedAt: time.Now().UTC(),
		})
}

func (r *Repository) FindByToken(
	ctx context.Context,
	hash [32]byte,
) (User, error) {
	var user User

	err := builder.Query(r.db, ctx).
		Table("users u").
		Select("u.id", "u.name", "u.telegram").
		InnerJoin("auth_tokens t", "u.id", "=", "t.user_id").
		Where("t.token_hash", "=", hash[:]).
		Where("t.expires_at", ">", expression.Raw("UTC_TIMESTAMP(6)")).
		First(&user)

	return user, err
}

func (r *Repository) DeleteToken(
	ctx context.Context,
	hash [32]byte,
) error {
	err := builder.Query(r.db, ctx).
		Table("auth_tokens").
		Where("token_hash", "=", hash[:]).
		Delete()

	return err
}
