package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/musiermoore/oksana-vpn-api/internal/referral"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenCreation      = errors.New("failed to create token")
	ErrLoginTaken         = errors.New("login is already in use")
)

type Service struct {
	repo      repositoryStore
	referrals *referral.Service
}

type repositoryStore interface {
	FindByTelegram(context.Context, string) (User, error)
	FindByLogin(context.Context, string) (User, error)
	SetCredentials(context.Context, int64, string, string) error
	EnsureUUID(context.Context, int64) error
	FindUserByToken(context.Context, string) (User, error)
	UpdateLastUsedAt(context.Context, string) error
	DeleteToken(context.Context, string) error
	FindOrCreateByTelegramID(context.Context, string, string, *string) (User, error)
	CreateToken(context.Context, int64, string, time.Time, TokenSource) error
	CreateOIDCTransaction(context.Context, string, oidcTransaction) error
	ConsumeOIDCTransaction(context.Context, string, time.Time) (oidcTransaction, error)
	ResolveOIDCUser(context.Context, string, string, string, string) (User, error)
	CreateOIDCLoginTicket(context.Context, string, int64, time.Time) error
	ConsumeOIDCLoginTicket(context.Context, string, time.Time) (int64, error)
}

func NewService(repo repositoryStore, referrals *referral.Service) *Service {
	return &Service{repo: repo, referrals: referrals}
}

func tokenHash(token string) (string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(raw) != 32 {
		return "", ErrInvalidToken
	}
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:]), nil
}

func (s *Service) Login(ctx context.Context, telegram, password string) (string, time.Time, error) {
	user, err := s.repo.FindByLogin(ctx, strings.ToLower(strings.TrimSpace(telegram)))
	if errors.Is(err, sql.ErrNoRows) {
		// Keep accounts created before login credentials were introduced able to log in.
		user, err = s.repo.FindByTelegram(ctx, telegram)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return "", time.Time{}, ErrInvalidCredentials
	}
	if err != nil {
		return "", time.Time{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash.String), []byte(password)); err != nil {
		return "", time.Time{}, ErrInvalidCredentials
	}
	token, expiresAt, err := s.CreateToken(ctx, user.ID, TokenSourceWeb)
	if err != nil {
		return "", time.Time{}, ErrTokenCreation
	}
	return token, expiresAt, nil
}

func (s *Service) SetCredentials(ctx context.Context, userID int64, login, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.repo.SetCredentials(ctx, userID, login, string(hash)); err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return ErrLoginTaken
		}
		return err
	}
	return nil
}

// Authenticate validates a token and returns the identity used by protected routes.
func (s *Service) Authenticate(ctx context.Context, token string) (int64, error) {
	hash, err := tokenHash(token)
	if err != nil {
		return 0, err
	}
	user, err := s.repo.FindUserByToken(ctx, hash)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrInvalidToken
	}
	if err != nil {
		return 0, err
	}
	if user.UUID == nil || strings.TrimSpace(*user.UUID) == "" {
		if err := s.repo.EnsureUUID(ctx, user.ID); err != nil {
			return 0, err
		}
	}
	if err := s.repo.UpdateLastUsedAt(ctx, hash); err != nil {
		log.Printf("WARNING: last_used_at failed to update: %v", err)
	}
	return user.ID, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	hash, err := tokenHash(token)
	if err != nil {
		return err
	}
	return s.repo.DeleteToken(ctx, hash)
}

func (s *Service) GetOrCreateUserByTelegramID(ctx context.Context, telegramID, firstName, lastName, username string) (User, error) {
	username = strings.TrimPrefix(strings.TrimSpace(username), "@")
	name := strings.Join(strings.Fields(firstName+" "+lastName), " ")
	if name == "" {
		name = username
	}
	if name == "" {
		name = telegramID
	}

	var telegram *string
	if username != "" {
		value := "@" + username
		telegram = &value
	}

	return s.repo.FindOrCreateByTelegramID(ctx, telegramID, name, telegram)
}

func (s *Service) AttachReferral(ctx context.Context, user User, startParam *string) error {
	return s.referrals.AttachReferral(ctx, referral.Participant{
		UserID:                     user.ID,
		ReferrerID:                 user.ReferrerID,
		JoinedAt:                   user.JoinAt,
		AccumulatedDiscountPercent: user.ReferralAccumulatedDiscountPercent,
	}, startParam)
}

func (s *Service) CreateToken(ctx context.Context, userID int64, source TokenSource) (string, time.Time, error) {
	if err := s.repo.EnsureUUID(ctx, userID); err != nil {
		return "", time.Time{}, err
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", time.Time{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	hash, err := tokenHash(token)
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)
	if err := s.repo.CreateToken(ctx, userID, hash, expiresAt, source); err != nil {
		return "", time.Time{}, err
	}
	return token, expiresAt, nil
}
