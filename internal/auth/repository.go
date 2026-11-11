package auth

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
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

func (r *Repository) FindByLogin(ctx context.Context, login string) (User, error) {
	var user User
	err := builder.Query(r.db, ctx).
		Table(usersTable).
		Select("id", "password").
		Where("login", "=", login).
		First(&user)
	return user, err
}

func (r *Repository) SetCredentials(ctx context.Context, userID int64, login, passwordHash string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET login = ?, password = ?, updated_at = UTC_TIMESTAMP(6) WHERE id = ?`,
		login, passwordHash, userID,
	)
	return err
}

func (r *Repository) FindOrCreateByTelegramID(ctx context.Context, id, name string, telegram *string) (User, error) {
	user, err := r.FindByTelegramID(ctx, id)
	if err == nil {
		return r.ensureUUID(ctx, user)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return User{}, err
	}

	now := time.Now().UTC()
	generatedUUID := uuid.New().String()
	newUser := User{
		UUID:       &generatedUUID,
		Name:       name,
		Telegram:   telegram,
		TelegramID: &id,
		JoinAt:     &now,
		UpdatedAt:  &now,
	}
	if err := builder.Query(r.db, ctx).Table(usersTable).Insert(
		newUser,
		"uuid",
		"name",
		"telegram",
		"telegram_id",
		"join_at",
		"updated_at",
	); err != nil {
		// Another request may have created the account after the initial lookup.
		user, lookupErr := r.FindByTelegramID(ctx, id)
		if lookupErr != nil {
			return User{}, err
		}
		return r.ensureUUID(ctx, user)
	}

	return r.FindByTelegramID(ctx, id)
}

func (r *Repository) ensureUUID(ctx context.Context, user User) (User, error) {
	if user.UUID != nil && *user.UUID != "" {
		return user, nil
	}

	generatedUUID := uuid.New().String()
	if err := builder.Query(r.db, ctx).
		Table(usersTable).
		Where("id", "=", user.ID).
		Where("uuid", "IS", expression.Raw("NULL")).
		Update(User{UUID: &generatedUUID}, "uuid"); err != nil {
		return User{}, err
	}
	if err := builder.Query(r.db, ctx).
		Table(usersTable).
		Where("id", "=", user.ID).
		Where("uuid", "=", "").
		Update(User{UUID: &generatedUUID}, "uuid"); err != nil {
		return User{}, err
	}

	var result User
	if err := builder.Query(r.db, ctx).
		Table(usersTable).
		Select("uuid").
		Where("id", "=", user.ID).
		First(&result); err != nil {
		return User{}, err
	}
	if result.UUID == nil || *result.UUID == "" {
		return User{}, sql.ErrNoRows
	}
	user.UUID = result.UUID
	return user, nil
}

func (r *Repository) FindByTelegramID(ctx context.Context, id string) (User, error) {
	var user User
	err := builder.Query(r.db, ctx).
		Table(usersTable).
		Select(
			"id",
			"uuid",
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

func (r *Repository) CreateToken(ctx context.Context, userID int64, hash string, expiresAt time.Time, source TokenSource) error {
	now := time.Now().UTC()
	return builder.Query(r.db, ctx).
		Table(authTokensTable).
		Insert(AuthToken{
			TokenHash:  hash,
			UserID:     userID,
			Source:     source,
			ExpiresAt:  expiresAt.UTC(),
			LastUsedAt: now,
			CreatedAt:  now,
			UpdatedAt:  now,
		})
}

func (r *Repository) FindUserByToken(ctx context.Context, hash string) (User, error) {
	var user User
	err := builder.Query(r.db, ctx).
		Table(authTokensTable+" t").
		Select("u.id", "u.uuid").
		InnerJoin(usersTable+" u", "u.id", "=", "t.user_id").
		Where("t.token_hash", "=", hash).
		Where("t.expires_at", ">", expression.Raw("UTC_TIMESTAMP(6)")).
		First(&user)
	return user, err
}

func (r *Repository) EnsureUUID(ctx context.Context, userID int64) error {
	var user User
	if err := builder.Query(r.db, ctx).
		Table(usersTable).
		Select("id", "uuid").
		Where("id", "=", userID).
		First(&user); err != nil {
		return err
	}
	_, err := r.ensureUUID(ctx, user)
	return err
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

func (r *Repository) CreateOIDCTransaction(ctx context.Context, stateHash string, tx oidcTransaction) error {
	_, _ = r.db.ExecContext(ctx, `DELETE FROM telegram_oidc_transactions WHERE expires_at <= UTC_TIMESTAMP(6) ORDER BY expires_at LIMIT 1000`)
	_, err := r.db.ExecContext(ctx, `INSERT INTO telegram_oidc_transactions
		(state_hash, nonce, pkce_verifier, return_url, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, UTC_TIMESTAMP(6))`, stateHash, tx.Nonce, tx.Verifier, tx.ReturnURL, tx.ExpiresAt.UTC())
	return err
}

func (r *Repository) ConsumeOIDCTransaction(ctx context.Context, stateHash string, now time.Time) (oidcTransaction, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return oidcTransaction{}, err
	}
	defer tx.Rollback()
	var result oidcTransaction
	err = tx.QueryRowContext(ctx, `SELECT nonce, pkce_verifier, return_url, expires_at
		FROM telegram_oidc_transactions WHERE state_hash = ? AND expires_at > ? FOR UPDATE`, stateHash, now.UTC()).Scan(
		&result.Nonce, &result.Verifier, &result.ReturnURL, &result.ExpiresAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return oidcTransaction{}, ErrOIDCTransaction
		}
		return oidcTransaction{}, err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM telegram_oidc_transactions WHERE state_hash = ?`, stateHash); err != nil {
		return oidcTransaction{}, err
	}
	if err = tx.Commit(); err != nil {
		return oidcTransaction{}, err
	}
	return result, nil
}

func (r *Repository) ResolveOIDCUser(ctx context.Context, subject, telegramID, name, username string) (User, error) {
	if strings.TrimSpace(subject) == "" {
		return User{}, ErrOIDCIdentity
	}
	name = strings.TrimSpace(name)
	username = strings.TrimPrefix(strings.TrimSpace(username), "@")
	var telegram *string
	if username != "" {
		value := "@" + username
		telegram = &value
	}
	if name == "" {
		name = username
	}
	if name == "" {
		name = "Telegram user"
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback()
	now := time.Now().UTC()

	var mapped User
	err = scanOIDCUser(tx.QueryRowContext(ctx, `SELECT u.id, u.uuid, u.name, u.telegram, u.telegram_id, u.referrer_id, u.join_at, u.referral_accumulated_discount_percent
		FROM users u INNER JOIN telegram_oidc_identities i ON i.user_id = u.id WHERE i.subject = ? FOR UPDATE`, subject), &mapped)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return User{}, err
	}
	if err == nil {
		if telegramID == "" {
			return mapped, tx.Commit()
		}
		if mapped.TelegramID != nil && *mapped.TelegramID != "" {
			if *mapped.TelegramID != telegramID {
				return User{}, ErrOIDCIdentityConflict
			}
			return mapped, tx.Commit()
		}
	}

	if telegramID != "" {
		matches, findErr := findOIDCUsersByTelegramID(ctx, tx, telegramID)
		if findErr != nil {
			return User{}, findErr
		}
		if len(matches) > 1 {
			return User{}, ErrOIDCIdentityConflict
		}
		if len(matches) == 1 {
			target := matches[0]
			if err == nil && target.ID != mapped.ID {
				if err := ensureUserHasNoOtherOIDCIdentity(ctx, tx, target.ID, subject); err != nil {
					return User{}, err
				}
				if _, err := tx.ExecContext(ctx, `UPDATE telegram_oidc_identities SET user_id = ? WHERE subject = ? AND user_id = ?`, target.ID, subject, mapped.ID); err != nil {
					return User{}, ErrOIDCIdentityConflict
				}
				// Revoke the session issued for the temporary OIDC-only account.
				if _, err := tx.ExecContext(ctx, `DELETE FROM `+authTokensTable+` WHERE user_id = ?`, mapped.ID); err != nil {
					return User{}, err
				}
			} else if err != nil {
				if err := ensureUserHasNoOtherOIDCIdentity(ctx, tx, target.ID, subject); err != nil {
					return User{}, err
				}
				if _, err := tx.ExecContext(ctx, `INSERT INTO telegram_oidc_identities (subject, user_id, created_at) VALUES (?, ?, ?)`, subject, target.ID, now); err != nil {
					return User{}, ErrOIDCIdentityConflict
				}
			}
			if err := tx.Commit(); err != nil {
				return User{}, err
			}
			return target, nil
		}
	}

	if err == nil {
		// This OIDC subject is already mapped to its own account and has no legacy-ID match.
		if telegramID != "" {
			if _, err := tx.ExecContext(ctx, `UPDATE users SET telegram_id = ?, telegram = COALESCE(telegram, ?), updated_at = ? WHERE id = ? AND (telegram_id IS NULL OR telegram_id = '')`, telegramID, telegram, now, mapped.ID); err != nil {
				return User{}, err
			}
			mapped.TelegramID = &telegramID
			if mapped.Telegram == nil {
				mapped.Telegram = telegram
			}
		}
		if err := tx.Commit(); err != nil {
			return User{}, err
		}
		return mapped, nil
	}

	generatedUUID := uuid.New().String()
	var telegramIDValue any
	if telegramID != "" {
		telegramIDValue = telegramID
	}
	res, err := tx.ExecContext(ctx, `INSERT INTO users (uuid, name, telegram, telegram_id, join_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, generatedUUID, name, telegram, telegramIDValue, now, now, now)
	if err != nil {
		return User{}, err
	}
	userID, err := res.LastInsertId()
	if err != nil {
		return User{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO telegram_oidc_identities (subject, user_id, created_at) VALUES (?, ?, ?)`, subject, userID, now); err != nil {
		return User{}, ErrOIDCIdentityConflict
	}
	if err = tx.Commit(); err != nil {
		return User{}, err
	}
	var telegramIDPtr *string
	if telegramID != "" {
		telegramIDPtr = &telegramID
	}
	return User{ID: userID, UUID: &generatedUUID, Name: name, Telegram: telegram, TelegramID: telegramIDPtr, JoinAt: &now}, nil
}

type rowScanner interface{ Scan(dest ...any) error }

func scanOIDCUser(scanner rowScanner, user *User) error {
	return scanner.Scan(&user.ID, &user.UUID, &user.Name, &user.Telegram, &user.TelegramID, &user.ReferrerID, &user.JoinAt, &user.ReferralAccumulatedDiscountPercent)
}

func findOIDCUsersByTelegramID(ctx context.Context, tx *sql.Tx, telegramID string) ([]User, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id, uuid, name, telegram, telegram_id, referrer_id, join_at, referral_accumulated_discount_percent
		FROM users WHERE telegram_id = ? ORDER BY id LIMIT 2 FOR UPDATE`, telegramID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := make([]User, 0, 2)
	for rows.Next() {
		var user User
		if err := scanOIDCUser(rows, &user); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func ensureUserHasNoOtherOIDCIdentity(ctx context.Context, tx *sql.Tx, userID int64, subject string) error {
	var existing string
	err := tx.QueryRowContext(ctx, `SELECT subject FROM telegram_oidc_identities WHERE user_id = ? FOR UPDATE`, userID).Scan(&existing)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if existing != subject {
		return ErrOIDCIdentityConflict
	}
	return nil
}

func (r *Repository) CreateOIDCLoginTicket(ctx context.Context, ticketHash string, userID int64, expiresAt time.Time) error {
	_, _ = r.db.ExecContext(ctx, `DELETE FROM telegram_oidc_login_tickets WHERE expires_at <= UTC_TIMESTAMP(6) ORDER BY expires_at LIMIT 1000`)
	_, err := r.db.ExecContext(ctx, `INSERT INTO telegram_oidc_login_tickets (ticket_hash, user_id, expires_at, created_at)
		VALUES (?, ?, ?, UTC_TIMESTAMP(6))`, ticketHash, userID, expiresAt.UTC())
	return err
}

func (r *Repository) ConsumeOIDCLoginTicket(ctx context.Context, ticketHash string, now time.Time) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var userID int64
	if err := tx.QueryRowContext(ctx, `SELECT user_id FROM telegram_oidc_login_tickets WHERE ticket_hash = ? AND expires_at > ? FOR UPDATE`, ticketHash, now.UTC()).Scan(&userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrOIDCTicket
		}
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM telegram_oidc_login_tickets WHERE ticket_hash = ?`, ticketHash); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return userID, nil
}
