package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func tokenHash(token string) ([32]byte, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)

	if err != nil || len(raw) != 32 {
		return [32]byte{}, ErrInvalidToken
	}

	return sha256.Sum256(raw), nil
}

func (s *Service) Login(
	ctx context.Context,
	telegram string,
	password string,
) (string, time.Time, error) {
	user, err := s.repo.FindByTelegram(ctx, telegram)

	if errors.Is(err, sql.ErrNoRows) {
		return "", time.Time{}, ErrInvalidCredentials
	}

	if err != nil {
		return "", time.Time{}, err
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(password),
	)

	if err != nil {
		return "", time.Time{}, ErrInvalidCredentials
	}

	// Generate 32 cryptographically secure random bytes.
	raw := make([]byte, 32)

	if _, err := rand.Read(raw); err != nil {
		return "", time.Time{}, err
	}

	token := base64.RawURLEncoding.EncodeToString(raw)

	// Never store the original token.
	hash := sha256.Sum256(raw)

	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)

	err = s.repo.CreateToken(
		ctx,
		user.ID,
		hash,
		expiresAt,
	)

	if err != nil {
		return "", time.Time{}, err
	}

	return token, expiresAt, nil
}

func (s *Service) Authenticate(
	ctx context.Context,
	token string,
) (User, error) {
	hash, err := tokenHash(token)

	if err != nil {
		return User{}, err
	}

	user, err := s.repo.FindByToken(ctx, hash)

	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrInvalidToken
	}

	return user, err
}

func (s *Service) Logout(
	ctx context.Context,
	token string,
) error {
	hash, err := tokenHash(token)

	if err != nil {
		return err
	}

	return s.repo.DeleteToken(ctx, hash)
}
