package vless

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

type linkUser struct {
	UUID              string `db:"uuid"`
	IsHealthCheckUser bool   `db:"is_health_check_user"`
	HasAccess         bool
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindByID(ctx context.Context, userID int64) (linkUser, error) {
	var user linkUser
	err := builder.Query(r.db, ctx).
		Table("users u").
		Select("u.uuid", "u.is_health_check_user").
		Where("u.id", "=", userID).
		Where("u.deleted_at", "IS", expression.Raw("NULL")).
		First(&user)

	if err != nil {
		return user, err
	}

	if user.IsHealthCheckUser {
		user.HasAccess = true
		return user, nil
	}

	var activeSubscription struct {
		ID int64 `db:"id"`
	}
	today := time.Now().UTC().Format("2006-01-02")
	err = builder.Query(r.db, ctx).
		Table("user_subscriptions").
		Select("id").
		Where("user_id", "=", userID).
		Where("start_date", "<=", today).
		Where("end_date", ">=", today).
		First(&activeSubscription)

	if err == sql.ErrNoRows {
		return user, nil
	}

	if err != nil {
		return user, err
	}

	user.HasAccess = true
	return user, nil
}
