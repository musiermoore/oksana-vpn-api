package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenCreation      = errors.New("failed to create token")
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func tokenHash(token string) (string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)

	if err != nil || len(raw) != 32 {
		return "", ErrInvalidToken
	}

	sum := sha256.Sum256([]byte(token))

	return hex.EncodeToString(sum[:]), nil
}

func (s *Service) Login(
	ctx context.Context,
	telegram string,
	password string,
) (string, time.Time, error) {
	user, err := s.GetUserByTelegram(ctx, telegram)

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

	token, expiresAt, err := s.CreateToken(ctx, user)

	if err != nil {
		return "", time.Time{}, ErrTokenCreation
	}

	return token, expiresAt, nil
}

func (s *Service) Authenticate(
	ctx context.Context,
	token string,
) (User, error) {
	hash, err := tokenHash(token)

	if err != nil {
		fmt.Println(fmt.Sprintf("Hash: %s. Err: %s", err), hash)
		return User{}, err
	}

	user, err := s.repo.FindByToken(ctx, hash)

	if errors.Is(err, sql.ErrNoRows) {
		fmt.Println(fmt.Sprintf("No user: %s. Err: %s", hash, err))
		return User{}, ErrInvalidToken
	}

	err = s.repo.UpdateLastUsedAt(ctx, hash)

	if err != nil {
		fmt.Println(fmt.Errorf("WARNING: last_used_at failed to update. Error: %s", err))
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

func (service *Service) GetUserByTelegram(
	ctx context.Context,
	telegram string,
) (User, error) {
	return service.repo.FindByTelegram(ctx, telegram)
}

func (service *Service) CreateToken(
	ctx context.Context,
	user User,
) (string, time.Time, error) {
	// Generate 32 cryptographically secure random bytes.
	raw := make([]byte, 32)

	if _, err := rand.Read(raw); err != nil {
		return "", time.Time{}, err
	}

	token := base64.RawURLEncoding.EncodeToString(raw)

	// Never store the original token.
	hash := sha256.Sum256([]byte(token))
	storedHash := hex.EncodeToString(hash[:])

	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)

	err := service.repo.CreateToken(
		ctx,
		user.ID,
		storedHash,
		expiresAt,
	)

	if err != nil {
		return "", time.Time{}, err
	}

	return token, expiresAt, nil
}
