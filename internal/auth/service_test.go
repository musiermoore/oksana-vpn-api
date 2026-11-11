package auth

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"testing"
	"time"
)

type fakeRepositoryStore struct {
	userID            int64
	ensureUUIDCalled  bool
	ensureBeforeToken bool
	updateLastUsed    bool
	createTokenCalled bool
	ensureErr         error
}

func (r *fakeRepositoryStore) FindByTelegram(context.Context, string) (User, error) {
	return User{}, sql.ErrNoRows
}

func (r *fakeRepositoryStore) FindByLogin(context.Context, string) (User, error) {
	return User{}, sql.ErrNoRows
}

func (r *fakeRepositoryStore) SetCredentials(context.Context, int64, string, string) error {
	return nil
}

func (r *fakeRepositoryStore) EnsureUUID(_ context.Context, userID int64) error {
	r.ensureUUIDCalled = true
	r.userID = userID
	return r.ensureErr
}

func (r *fakeRepositoryStore) FindUserByToken(context.Context, string) (User, error) {
	return User{ID: 42}, nil
}

func (r *fakeRepositoryStore) UpdateLastUsedAt(context.Context, string) error {
	r.updateLastUsed = true
	return nil
}

func (r *fakeRepositoryStore) DeleteToken(context.Context, string) error {
	return nil
}

func (r *fakeRepositoryStore) FindOrCreateByTelegramID(context.Context, string, string, *string) (User, error) {
	return User{}, nil
}

func (r *fakeRepositoryStore) CreateToken(context.Context, int64, string, time.Time, TokenSource) error {
	r.createTokenCalled = true
	r.ensureBeforeToken = r.ensureUUIDCalled
	return nil
}

func (r *fakeRepositoryStore) CreateOIDCTransaction(context.Context, string, oidcTransaction) error {
	return nil
}

func (r *fakeRepositoryStore) ConsumeOIDCTransaction(context.Context, string, time.Time) (oidcTransaction, error) {
	return oidcTransaction{}, nil
}

func (r *fakeRepositoryStore) ResolveOIDCUser(context.Context, string, string, string, string) (User, error) {
	return User{}, nil
}

func (r *fakeRepositoryStore) CreateOIDCLoginTicket(context.Context, string, int64, time.Time) error {
	return nil
}

func (r *fakeRepositoryStore) ConsumeOIDCLoginTicket(context.Context, string, time.Time) (int64, error) {
	return 0, nil
}

func TestCreateTokenEnsuresUUIDBeforeIssuingToken(t *testing.T) {
	repository := &fakeRepositoryStore{}
	service := NewService(repository, nil)

	if _, _, err := service.CreateToken(context.Background(), 42, TokenSourceWeb); err != nil {
		t.Fatalf("CreateToken error=%v", err)
	}
	if !repository.ensureUUIDCalled || repository.userID != 42 {
		t.Fatalf("EnsureUUID called=%t for user %d; want user 42", repository.ensureUUIDCalled, repository.userID)
	}
	if !repository.createTokenCalled || !repository.ensureBeforeToken {
		t.Fatal("token was not persisted after ensuring UUID")
	}
}

func TestAuthenticateEnsuresUUIDForExistingToken(t *testing.T) {
	repository := &fakeRepositoryStore{}
	service := NewService(repository, nil)
	token := base64.RawURLEncoding.EncodeToString(make([]byte, 32))

	userID, err := service.Authenticate(context.Background(), token)
	if err != nil {
		t.Fatalf("Authenticate error=%v", err)
	}
	if userID != 42 || !repository.ensureUUIDCalled || repository.userID != 42 {
		t.Fatalf("userID=%d, EnsureUUID called=%t for user %d; want 42 and ensured", userID, repository.ensureUUIDCalled, repository.userID)
	}
	if !repository.updateLastUsed {
		t.Fatal("token last-used timestamp was not updated")
	}
}

func TestCreateTokenDoesNotIssueTokenWhenUUIDEnsureFails(t *testing.T) {
	wantErr := errors.New("uuid storage failed")
	repository := &fakeRepositoryStore{ensureErr: wantErr}
	service := NewService(repository, nil)

	if _, _, err := service.CreateToken(context.Background(), 42, TokenSourceTelegram); !errors.Is(err, wantErr) {
		t.Fatalf("CreateToken error=%v; want %v", err, wantErr)
	}
	if repository.createTokenCalled {
		t.Fatal("token was persisted despite UUID ensure failure")
	}
}
