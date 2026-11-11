package vless

import (
	"context"
	"errors"
	"net/url"
	"strings"
)

var ErrNoActiveAccess = errors.New("user has no active access")

type LinkResponse struct {
	HappDeepLink     string `json:"happ_deep_link"`
	IncyDeepLink     string `json:"incy_deep_link"`
	V2RayTunDeepLink string `json:"v2raytun_deep_link"`
	ShowRawLink      bool   `json:"show_raw_link"`
}

type Service struct {
	repository   linkDataRepository
	publicDomain string
}

type linkDataRepository interface {
	FindByID(ctx context.Context, userID int64) (linkUser, error)
}

func NewService(repository linkDataRepository, publicDomain string) *Service {
	return &Service{repository: repository, publicDomain: publicDomain}
}

func (s *Service) Links(ctx context.Context, userID int64) (LinkResponse, error) {
	user, err := s.repository.FindByID(ctx, userID)
	if err != nil {
		return LinkResponse{}, err
	}
	if !user.HasAccess {
		return LinkResponse{}, ErrNoActiveAccess
	}

	return BuildLinks(user.UUID, s.publicDomain), nil
}

func BuildLinks(userUUID, publicDomain string) LinkResponse {
	base := strings.TrimRight(strings.TrimSpace(publicDomain), "/") + "/deep-link/"
	token := url.QueryEscape(userUUID)
	return LinkResponse{
		HappDeepLink:     base + "happ?token=" + token,
		IncyDeepLink:     base + "incy?token=" + token,
		V2RayTunDeepLink: base + "v2raytun?token=" + token,
		ShowRawLink:      false,
	}
}
