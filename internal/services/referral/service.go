package referral

import (
	"context"
	"fmt"
	"os"
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

// Summary is the referral information returned with an authenticated user.
type Summary struct {
	ReferralLink               *string `json:"referral_link"`
	ReferralCode               string  `json:"referral_code"`
	HasReferrer                bool    `json:"has_referrer"`
	CanClaim                   bool    `json:"can_claim"`
	InvitedCount               int     `json:"invited_count"`
	ActiveReferralsCount       int     `json:"active_referrals_count"`
	AccumulatedDiscountPercent int     `json:"accumulated_discount_percent"`
	PermanentDiscountPercent   int     `json:"permanent_discount_percent"`
	TotalDiscountPercent       int     `json:"total_discount_percent"`
	NextLevelActiveReferrals   *int    `json:"next_level_active_referrals"`
	RemainingToNextLevel       int     `json:"remaining_to_next_level"`
}

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
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

func (s *Service) AttachReferral(ctx context.Context, userID int64, currentReferrerID *int64, startParam *string) error {
	referrerID := ParseReferralInput(startParam)
	if referrerID == nil || *referrerID == userID || currentReferrerID != nil {
		return nil
	}
	return s.repo.AttachReferral(ctx, userID, *referrerID)
}

func (s *Service) GetSummary(
	ctx context.Context,
	userID int64,
	referrerID *int64,
	joinAt *time.Time,
	accumulatedDiscountPercent int,
) (Summary, error) {
	invitedCount, err := s.repo.CountInvitedUsers(ctx, userID)
	if err != nil {
		return Summary{}, err
	}
	activeCount, err := s.repo.CountActiveReferrals(ctx, userID)
	if err != nil {
		return Summary{}, err
	}

	permanentPercent := ResolvePermanentDiscountPercent(activeCount)
	nextLevel := resolveNextLevel(activeCount)
	remaining := 0
	if nextLevel != nil {
		remaining = max(0, *nextLevel-activeCount)
	}

	code := fmt.Sprintf("ref_%d", userID)
	var link *string
	botUsername := strings.TrimLeft(strings.TrimSpace(os.Getenv("TELEGRAM_BOT_USERNAME")), "@")
	if botUsername != "" {
		value := fmt.Sprintf("https://t.me/%s?startapp=%s", botUsername, code)
		link = &value
	}

	return Summary{
		ReferralLink:               link,
		ReferralCode:               code,
		HasReferrer:                referrerID != nil,
		CanClaim:                   CanClaimManually(referrerID, joinAt, time.Now()),
		InvitedCount:               invitedCount,
		ActiveReferralsCount:       activeCount,
		AccumulatedDiscountPercent: accumulatedDiscountPercent,
		PermanentDiscountPercent:   permanentPercent,
		TotalDiscountPercent:       accumulatedDiscountPercent + permanentPercent,
		NextLevelActiveReferrals:   nextLevel,
		RemainingToNextLevel:       remaining,
	}, nil
}
