package account

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/musiermoore/oksana-vpn-api/internal/database/builder"
	"github.com/musiermoore/oksana-vpn-api/internal/database/builder/query/expression"
	"github.com/musiermoore/oksana-vpn-api/internal/database/builder/query/join"
)

const usersTable = "users"
const subscriptionCodesTable = "subscription_codes"

type profileRow struct {
	ID                                 int64      `db:"id"`
	UUID                               string     `db:"uuid"`
	Name                               string     `db:"name"`
	Telegram                           *string    `db:"telegram"`
	TelegramID                         *string    `db:"telegram_id"`
	Balance                            float64    `db:"balance"`
	Debt                               float64    `db:"debt"`
	IsAdmin                            bool       `db:"is_admin"`
	HasActiveAccess                    bool       `db:"has_active_access"`
	HasVlessWlConfigs                  bool       `db:"has_vless_wl_configs"`
	SubscriptionExpiresAt              *time.Time `db:"subscription_expires_at"`
	HasMoneyForNextSubscriptionMonth   bool       `db:"has_money_for_next_subscription_month"`
	ReferrerID                         *int64     `db:"referrer_id"`
	JoinAt                             *time.Time `db:"join_at"`
	ReferralAccumulatedDiscountPercent int        `db:"referral_accumulated_discount_percent"`
}

type subscriptionCodeRow struct {
	ID                int        `db:"id"`
	Code              string     `db:"code"`
	Months            *int       `db:"months"`
	Days              *int       `db:"days"`
	Price             float64    `db:"price"`
	Status            string     `db:"status"`
	ActivatedAt       *time.Time `db:"activated_at"`
	ExpiresAt         *time.Time `db:"expires_at"`
	ActivatedByUserID *int       `db:"activated_by_user_id"`
	CreatedAt         *time.Time `db:"created_at"`
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindByID(ctx context.Context, id int64) (profileRow, error) {
	var user profileRow
	err := builder.Query(r.db, ctx).
		Table(usersTable+" u").
		Select(
			"u.id",
			"u.uuid",
			"u.name",
			"u.telegram",
			"u.telegram_id",
			"u.balance",
			expression.Raw("GREATEST(0, -COALESCE(u.balance, 0)) AS debt"),
			"u.is_admin",
			expression.Raw("u.subscription_expires_at IS NOT NULL AND u.subscription_expires_at > UTC_TIMESTAMP() AS has_active_access"),
			expression.Raw("COUNT(vesc.id) > 0 AS has_vless_wl_configs"),
			expression.Raw(`u.subscription_expires_at IS NOT NULL
                AND u.subscription_expires_at > DATE_ADD(UTC_TIMESTAMP(), INTERVAL 1 MONTH) AS has_money_for_next_subscription_month`),
			"u.subscription_expires_at",
			"u.referrer_id",
			"u.join_at",
			"u.referral_accumulated_discount_percent",
		).
		LeftJoinGroup("vless_external_subscriptions ves", func(j *join.Builder) {
			j.Where("ves.is_active", "=", true)
			j.Where("ves.include_in_whitelist", "=", true)
			j.OnGroup(func(j *join.Builder) {
				j.Where("u.is_admin", "=", true)
				j.OrWhere("ves.is_ready", "=", true)
			})
			j.OnGroup(func(j *join.Builder) {
				j.Where("ves.is_free", "=", true)
				j.OrWhere("u.subscription_expires_at", ">", expression.Raw("UTC_TIMESTAMP()"))
			})
		}).
		LeftJoin("vless_external_subscription_configs vesc", "vesc.vless_external_subscription_id", "=", "ves.id").
		Where("u.id", "=", id).
		GroupBy("u.id").
		First(&user)
	return user, err
}

func (r *Repository) FindPurchasedSubscriptionCodes(ctx context.Context, userID int64) ([]SubscriptionCode, error) {
	rows := make([]subscriptionCodeRow, 0, 10)
	err := builder.Query(r.db, ctx).
		Table(subscriptionCodesTable).
		Select(
			"id",
			"code",
			"months",
			"days",
			"price",
			"status",
			"activated_at",
			"expires_at",
			"activated_by_user_id",
			"created_at",
		).
		Where("buyer_user_id", "=", userID).
		OrderBy("created_at", false).
		Limit(10).
		List(&rows)
	if err != nil {
		return nil, err
	}

	codes := make([]SubscriptionCode, 0, len(rows))
	for _, row := range rows {
		codes = append(codes, SubscriptionCode{
			ID:                row.ID,
			Code:              row.Code,
			Months:            row.Months,
			Days:              row.Days,
			Price:             row.Price,
			Status:            row.Status,
			ActivatedAt:       row.ActivatedAt,
			ExpiresAt:         row.ExpiresAt,
			ActivatedByUserID: row.ActivatedByUserID,
			CreatedAt:         row.CreatedAt,
		})
	}
	return codes, nil
}

func (r *Repository) SetUUID(ctx context.Context, id int64) error {
	return builder.Query(r.db, ctx).
		Table(usersTable).
		Where("id", "=", id).
		Update(struct {
			UUID string `db:"uuid"`
		}{
			UUID: uuid.New().String(),
		}, "uuid")
}
