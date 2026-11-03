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
	repo      *Repository
	referrals *referral.Service
}

func NewService(repo *Repository, referrals *referral.Service) *Service {
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
	userID, err := s.repo.FindUserIDByToken(ctx, hash)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrInvalidToken
	}
	if err != nil {
		return 0, err
	}
	if err := s.repo.UpdateLastUsedAt(ctx, hash); err != nil {
		log.Printf("WARNING: last_used_at failed to update: %v", err)
	}
	return userID, nil
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
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", time.Time{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(token))
	storedHash := hex.EncodeToString(hash[:])
	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)
	if err := s.repo.CreateToken(ctx, userID, storedHash, expiresAt, source); err != nil {
		return "", time.Time{}, err
	}
	return token, expiresAt, nil
}
