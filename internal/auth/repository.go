package auth

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/musiermoore/oksana-vpn-api/internal/database/builder"
	"github.com/musiermoore/oksana-vpn-api/internal/database/builder/query/expression"
	"github.com/musiermoore/oksana-vpn-api/internal/database/builder/query/join"
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
		Table(USERS_TABLE).
		Select("id", "name", "telegram", "password").
		Where("telegram", "=", "@"+telegram).
		First(&user)

	return user, err
}

func (r *Repository) FindByTelegramId(
	ctx context.Context,
	id string,
) (User, error) {
	var user User

	err := builder.Query(r.db, ctx).
		Table(USERS_TABLE).
		Select(
			"id",
			"name",
			"telegram",
			"telegram_id",
		).
		Where("telegram_id", "=", id).
		First(&user)

	return user, err
}

func (r *Repository) CreateToken(
	ctx context.Context,
	userID int64,
	hash string,
	expiresAt time.Time,
) error {
	return builder.Query(r.db, ctx).
		Table(AUTH_TOKENS_TABLE).
		Insert(AuthToken{
			TokenHash:  hash,
			UserId:     userID,
			ExpiresAt:  expiresAt.UTC(),
			LastUsedAt: time.Now().UTC(),
			CreatedAt:  time.Now().UTC(),
			UpdatedAt:  time.Now().UTC(),
		})
}

func (r *Repository) FindByToken(
	ctx context.Context,
	hash string,
) (User, error) {
	var user User

	err := builder.Query(r.db, ctx).
		Table(fmt.Sprintf("%s u", USERS_TABLE)).
		Select(
			"u.id",
			"u.name",
			"u.telegram",
			"u.telegram_id",
			"u.balance",
			expression.Raw("GREATEST(0, -COALESCE(u.balance, 0)) AS debt"),
			"u.is_admin",
			expression.Raw("u.subscription_expires_at IS NOT NULL AND u.subscription_expires_at > UTC_TIMESTAMP() AS has_active_access"),
			expression.Raw("COUNT(IFNULL(vesc.id, 0)) > 0 AS has_vless_wl_configs"),
			expression.Raw(
				`u.subscription_expires_at IS NOT NULL 
				AND u.subscription_expires_at > DATE_ADD(UTC_TIMESTAMP(), INTERVAL 1 MONTH) AS has_money_for_next_subscription_month`,
			),
			"u.subscription_expires_at",
		).
		InnerJoin(fmt.Sprintf("%s t", AUTH_TOKENS_TABLE), "u.id", "=", "t.user_id").
		LeftJoinGroup("vless_external_subscriptions ves", func(j *join.Builder) {
			j.Where("ves.is_active", "=", true)
			j.Where("ves.include_in_whitelist", "=", true)
			j.OnGroup(func(j *join.Builder) {
				j.On("u.is_admin", "=", "1")
				j.OrOn("ves.is_ready", "=", "1")
			})
			j.OnGroup(func(j *join.Builder) {
				j.Where("ves.is_free", "=", true)
				j.OrWhere("u.subscription_expires_at", ">", expression.Raw("UTC_TIMESTAMP()"))
			})
		}).
		LeftJoin("vless_external_subscription_configs vesc", "vesc.vless_external_subscription_id", "=", "ves.id").
		Where("t.token_hash", "=", hash).
		Where("t.expires_at", ">", expression.Raw("UTC_TIMESTAMP(6)")).
		GroupBy("u.id").
		First(&user)

	return user, err
}

func (r *Repository) DeleteToken(
	ctx context.Context,
	hash string,
) error {
	err := builder.Query(r.db, ctx).
		Table(AUTH_TOKENS_TABLE).
		Where("token_hash", "=", hash).
		Delete()

	return err
}

func (r *Repository) UpdateLastUsedAt(ctx context.Context, hash string) error {
	err := builder.Query(r.db, ctx).
		Table(AUTH_TOKENS_TABLE).
		Where("token_hash", "=", hash).
		Update(AuthToken{
			LastUsedAt: time.Now().UTC(),
			UpdatedAt:  time.Now().UTC(),
		}, "last_used_at", "updated_at")

	return err
}
