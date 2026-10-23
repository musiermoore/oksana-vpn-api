package referral

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var referralCodePattern = regexp.MustCompile(`^ref_(\d+)$`)
var referralInputPattern = regexp.MustCompile(`(?:start=|startapp=|tgWebAppStartParam=)(ref_\d+)`)

var permanentDiscountLevels = []struct {
	from    int
	percent int
}{
	{from: 40, percent: 20},
	{from: 20, percent: 15},
	{from: 10, percent: 10},
	{from: 5, percent: 5},
}

type Service struct {
	repo        *Repository
	botUsername string
}

func NewService(repo *Repository, botUsername string) *Service {
	return &Service{repo: repo, botUsername: botUsername}
}

func ParseReferrerID(startParam *string) *int64 {
	if startParam == nil {
		return nil
	}
	matches := referralCodePattern.FindStringSubmatch(strings.TrimSpace(*startParam))
	if len(matches) != 2 {
		return nil
	}

	var id int64
	if _, err := fmt.Sscan(matches[1], &id); err != nil || id <= 0 {
		return nil
	}
	return &id
}

func ParseReferralInput(value *string) *int64 {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if id := ParseReferrerID(&trimmed); id != nil {
		return id
	}
	matches := referralInputPattern.FindStringSubmatch(trimmed)
	if len(matches) != 2 {
		return nil
	}
	return ParseReferrerID(&matches[1])
}

func CanClaimManually(referrerID *int64, joinAt *time.Time, now time.Time) bool {
	if referrerID != nil || joinAt == nil {
		return false
	}
	return !joinAt.Before(oneMonthAgo(now))
}

func oneMonthAgo(value time.Time) time.Time {
	monthStart := time.Date(value.Year(), value.Month()-1, 1, value.Hour(), value.Minute(), value.Second(), value.Nanosecond(), value.Location())
	lastDay := time.Date(monthStart.Year(), monthStart.Month()+1, 0, 0, 0, 0, 0, value.Location()).Day()
	day := min(value.Day(), lastDay)
	return time.Date(monthStart.Year(), monthStart.Month(), day, value.Hour(), value.Minute(), value.Second(), value.Nanosecond(), value.Location())
}

func ResolvePermanentDiscountPercent(activeReferralsCount int) int {
	for _, level := range permanentDiscountLevels {
		if activeReferralsCount >= level.from {
			return level.percent
		}
	}
	return 0
}

func resolveNextLevel(activeReferralsCount int) *int {
	for i := len(permanentDiscountLevels) - 1; i >= 0; i-- {
		if activeReferralsCount < permanentDiscountLevels[i].from {
			next := permanentDiscountLevels[i].from
			return &next
		}
	}
	return nil
}

func (s *Service) AttachReferral(ctx context.Context, participant Participant, startParam *string) error {
	referrerID := ParseReferralInput(startParam)
	if referrerID == nil || *referrerID == participant.UserID || participant.ReferrerID != nil {
		return nil
	}
	return s.repo.AttachReferral(ctx, participant.UserID, *referrerID)
}

func (s *Service) GetSummary(
	ctx context.Context,
	participant Participant,
) (Summary, error) {
	invitedCount, err := s.repo.CountInvitedUsers(ctx, participant.UserID)
	if err != nil {
		return Summary{}, err
	}
	activeCount, err := s.repo.CountActiveReferrals(ctx, participant.UserID)
	if err != nil {
		return Summary{}, err
	}

	permanentPercent := ResolvePermanentDiscountPercent(activeCount)
	nextLevel := resolveNextLevel(activeCount)
	remaining := 0
	if nextLevel != nil {
		remaining = max(0, *nextLevel-activeCount)
	}

	code := fmt.Sprintf("ref_%d", participant.UserID)
	var link *string
	botUsername := strings.TrimLeft(strings.TrimSpace(s.botUsername), "@")
	if botUsername != "" {
		value := fmt.Sprintf("https://t.me/%s?startapp=%s", botUsername, code)
		link = &value
	}

	return Summary{
		ReferralLink:               link,
		ReferralCode:               code,
		HasReferrer:                participant.ReferrerID != nil,
		CanClaim:                   CanClaimManually(participant.ReferrerID, participant.JoinedAt, time.Now()),
		InvitedCount:               invitedCount,
		ActiveReferralsCount:       activeCount,
		AccumulatedDiscountPercent: participant.AccumulatedDiscountPercent,
		PermanentDiscountPercent:   permanentPercent,
		TotalDiscountPercent:       participant.AccumulatedDiscountPercent + permanentPercent,
		NextLevelActiveReferrals:   nextLevel,
		RemainingToNextLevel:       remaining,
	}, nil
}
