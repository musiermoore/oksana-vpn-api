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

func (r *Repository) FindByTelegram(ctx context.Context, telegram string) (User, error) {
	var user User
	err := builder.Query(r.db, ctx).
		Table(usersTable).
		Select(
			"id",
			"name",
			"telegram",
			"password",
		).
		Where("telegram", "=", "@"+telegram).
		First(&user)
	return user, err
}

func (r *Repository) FindByTelegramID(ctx context.Context, id string) (User, error) {
	var user User
	err := builder.Query(r.db, ctx).
		Table(usersTable).
		Select(
			"id",
			"name",
			"telegram",
			"telegram_id",
			"referrer_id",
			"join_at",
			"referral_accumulated_discount_percent",
		).
		Where("telegram_id", "=", id).
		First(&user)
	return user, err
}

func (r *Repository) CreateToken(ctx context.Context, userID int64, hash string, expiresAt time.Time) error {
	now := time.Now().UTC()
	return builder.Query(r.db, ctx).
		Table(authTokensTable).
		Insert(AuthToken{
			TokenHash:  hash,
			UserID:     userID,
			ExpiresAt:  expiresAt.UTC(),
			LastUsedAt: now,
			CreatedAt:  now,
			UpdatedAt:  now,
		})
}

func (r *Repository) FindUserIDByToken(ctx context.Context, hash string) (int64, error) {
	var result struct {
		ID int64 `db:"id"`
	}
	err := builder.Query(r.db, ctx).
		Table(authTokensTable+" t").
		Select("u.id").
		InnerJoin(usersTable+" u", "u.id", "=", "t.user_id").
		Where("t.token_hash", "=", hash).
		Where("t.expires_at", ">", expression.Raw("UTC_TIMESTAMP(6)")).
		First(&result)
	return result.ID, err
}

func (r *Repository) DeleteToken(ctx context.Context, hash string) error {
	return builder.Query(r.db, ctx).
		Table(authTokensTable).
		Where("token_hash", "=", hash).
		Delete()
}

func (r *Repository) UpdateLastUsedAt(ctx context.Context, hash string) error {
	now := time.Now().UTC()
	return builder.Query(r.db, ctx).
		Table(authTokensTable).
		Where("token_hash", "=", hash).
		Update(AuthToken{
			LastUsedAt: now,
			UpdatedAt:  now,
		}, "last_used_at", "updated_at")
}
