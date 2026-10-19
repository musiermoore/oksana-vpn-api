package referral

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/musiermoore/oksana-vpn-api/internal/database/builder"
	"github.com/musiermoore/oksana-vpn-api/internal/database/builder/query/expression"
)

const referralsTable = "referrals"
const usersTable = "users"

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CountInvitedUsers(ctx context.Context, userID int64) (int, error) {
	var result struct {
		Count int `db:"count"`
	}
	err := builder.Query(r.db, ctx).
		Table(referralsTable).
		Select(expression.Raw("COUNT(*) AS count")).
		Where("referrer_id", "=", userID).
		First(&result)
	return result.Count, err
}

func (r *Repository) CountActiveReferrals(ctx context.Context, userID int64) (int, error) {
	var result struct {
		Count int `db:"count"`
	}
	err := builder.Query(r.db, ctx).
		Table(referralsTable).
		Select(expression.Raw("COUNT(*) AS count")).
		Where("referrer_id", "=", userID).
		Where("qualifying_transaction_id", "IS NOT", expression.Raw("NULL")).
		First(&result)
	return result.Count, err
}

func (r *Repository) AttachReferral(ctx context.Context, userID, referrerID int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var user struct {
		ReferrerID sql.NullInt64 `db:"referrer_id"`
	}
	err = builder.Query(tx, ctx).
		Table(usersTable).
		Select("referrer_id").
		Where("id", "=", userID).
		ForUpdate().
		First(&user)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if user.ReferrerID.Valid || userID == referrerID {
		return nil
	}

	var activeReferrer struct {
		ID int64 `db:"id"`
	}
	err = builder.Query(tx, ctx).
		Table(usersTable).
		Select("id").
		Where("id", "=", referrerID).
		Where("subscription_expires_at", ">", expression.Raw("UTC_TIMESTAMP()")).
		ForUpdate().
		First(&activeReferrer)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	if err = builder.Query(tx, ctx).
		Table(usersTable).
		Where("id", "=", userID).
		Update(struct {
			ReferrerID int64     `db:"referrer_id"`
			UpdatedAt  time.Time `db:"updated_at"`
		}{ReferrerID: referrerID, UpdatedAt: time.Now().UTC()}, "referrer_id", "updated_at"); err != nil {
		return err
	}

	var existingReferral struct {
		ID int64 `db:"id"`
	}
	err = builder.Query(tx, ctx).
		Table(referralsTable).
		Select("id").
		Where("referral_user_id", "=", userID).
		First(&existingReferral)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if errors.Is(err, sql.ErrNoRows) {
		if err = builder.Query(tx, ctx).
			Table(referralsTable).
			Insert(struct {
				ReferralUserID int64     `db:"referral_user_id"`
				ReferrerID     int64     `db:"referrer_id"`
				CreatedAt      time.Time `db:"created_at"`
				UpdatedAt      time.Time `db:"updated_at"`
			}{ReferralUserID: userID, ReferrerID: referrerID, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}); err != nil {
			return err
		}
	}

	return tx.Commit()
}
