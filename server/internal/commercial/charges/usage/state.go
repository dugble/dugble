package usage

import "errors"

type Outcome string

const (
	OutcomeApplied                 Outcome = "applied"
	OutcomeAlreadyApplied          Outcome = "already_applied"
	OutcomeCreditApplied           Outcome = "credit_applied"
	OutcomeInsufficientBalance     Outcome = "insufficient_balance"
	OutcomeTeamNotFound            Outcome = "team_not_found"
	OutcomeTeamInactive            Outcome = "team_inactive"
	OutcomeUnsupportedMarket       Outcome = "unsupported_market"
	OutcomeWalletNotFound          Outcome = "wallet_not_found"
	OutcomeSubscriptionUnavailable Outcome = "subscription_unavailable"
	OutcomeRateNotFound            Outcome = "rate_not_found"
	OutcomeFXRateNotFound          Outcome = "fx_rate_not_found"
	OutcomeCurrencyMismatch        Outcome = "currency_mismatch"
	OutcomeAmountOverflow          Outcome = "amount_overflow"
)

var (
	ErrTeamNotFound            = errors.New("billing team not found")
	ErrTeamInactive            = errors.New("billing team is not active")
	ErrUnsupportedMarket       = errors.New("billing market is not supported")
	ErrWalletNotFound          = errors.New("team wallet not found")
	ErrSubscriptionUnavailable = errors.New("active subscription entitlement is unavailable")
	ErrRateNotFound            = errors.New("active product rate not found")
	ErrFXRateNotFound          = errors.New("active FX rate not found")
	ErrCurrencyMismatch        = errors.New("billing currency does not match team market")
	ErrInsufficientBalance     = errors.New("insufficient wallet balance")
	ErrAmountOverflow          = errors.New("billing amount exceeds supported range")
)
