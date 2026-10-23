package referral

import "time"

// Participant carries the user facts needed by referral rules.
type Participant struct {
	UserID                     int64
	ReferrerID                 *int64
	JoinedAt                   *time.Time
	AccumulatedDiscountPercent int
}

// Summary is the referral information returned with an account profile.
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
