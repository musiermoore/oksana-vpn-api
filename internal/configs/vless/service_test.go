package vless

import (
	"context"
	"errors"
	"testing"
)

type fakeLinkDataRepository struct {
	user linkUser
}

func (r *fakeLinkDataRepository) FindByID(context.Context, int64) (linkUser, error) {
	return r.user, nil
}

func TestServiceLinksRejectsUserWithoutActiveAccess(t *testing.T) {
	repository := &fakeLinkDataRepository{user: linkUser{UUID: "abc-123", HasAccess: false}}
	service := NewService(repository, "https://connect.example")

	if _, err := service.Links(context.Background(), 42); !errors.Is(err, ErrNoActiveAccess) {
		t.Fatalf("Links error=%v; want ErrNoActiveAccess", err)
	}
}

func TestServiceLinksBuildsLinksFromAuthenticatedUserUUID(t *testing.T) {
	repository := &fakeLinkDataRepository{
		user: linkUser{UUID: "abc-123", HasAccess: true},
	}
	service := NewService(repository, "https://connect.example")

	links, err := service.Links(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if links.HappDeepLink != "https://connect.example/deep-link/happ?token=abc-123" {
		t.Errorf("Happ link=%q", links.HappDeepLink)
	}
	if links.IncyDeepLink != "https://connect.example/deep-link/incy?token=abc-123" {
		t.Errorf("Incy link=%q", links.IncyDeepLink)
	}
	if links.V2RayTunDeepLink != "https://connect.example/deep-link/v2raytun?token=abc-123" {
		t.Errorf("V2RayTun link=%q", links.V2RayTunDeepLink)
	}
}
