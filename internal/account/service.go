package account

import (
	"context"
	"log"

	"github.com/musiermoore/oksana-vpn-api/internal/referral"
)

type Service struct {
	repo      *Repository
	referrals *referral.Service
}

func NewService(repo *Repository, referrals *referral.Service) *Service {
	return &Service{repo: repo, referrals: referrals}
}

func (s *Service) GetProfile(ctx context.Context, userID int64) (Profile, error) {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return Profile{}, err
	}
	if user.UUID == nil || *user.UUID == "" {
		userUUID, err := s.repo.EnsureUUID(ctx, user.ID)
		if err != nil {
			log.Printf("WARNING: uuid failed to set for user %d: %v", user.ID, err)
			return Profile{}, err
		}
		user.UUID = userUUID
	}

	profile := Profile{
		ID:                               user.ID,
		UUID:                             *user.UUID,
		Name:                             user.Name,
		Telegram:                         user.Telegram,
		TelegramID:                       user.TelegramID,
		Balance:                          user.Balance,
		Debt:                             user.Debt,
		IsAdmin:                          user.IsAdmin,
		HasActiveAccess:                  user.HasActiveAccess,
		HasVlessWlConfigs:                user.HasVlessWlConfigs,
		SubscriptionExpiresAt:            user.SubscriptionExpiresAt,
		HasMoneyForNextSubscriptionMonth: user.HasMoneyForNextSubscriptionMonth,
		ReferrerID:                       user.ReferrerID,
		JoinAt:                           user.JoinAt,
	}

	profile.Referral, err = s.referrals.GetSummary(ctx, referral.Participant{
		UserID:                     user.ID,
		ReferrerID:                 user.ReferrerID,
		JoinedAt:                   user.JoinAt,
		AccumulatedDiscountPercent: user.ReferralAccumulatedDiscountPercent,
	})
	if err != nil {
		return Profile{}, err
	}

	profile.SubscriptionCodes, err = s.repo.FindPurchasedSubscriptionCodes(ctx, user.ID)
	if err != nil {
		return Profile{}, err
	}

	return profile, nil
}
