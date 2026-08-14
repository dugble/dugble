package usage

import (
	"context"

	sentrymonitoring "github.com/dugble/dugble/server/internal/adapters/monitoring/sentry"
)

var (
	_ SMSBilling   = (*Service)(nil)
	_ EmailBilling = (*Service)(nil)
)

// ObserveCommittedCharge records a durable billing decision after its enclosing
// transaction has committed. It deliberately has no error result: message
// acceptance must not be reversed by a best-effort observation failure.
func (s *Service) ObserveCommittedCharge(ctx context.Context, committed CommittedCharge) {
	sentrymonitoring.InfoContext(ctx, "billing charge committed",
		"billing_channel", committed.Channel,
		"billing_settlement", committed.Settlement,
		"team_id", committed.TeamID,
		"message_id", committed.MessageID,
		"billing_outcome", committed.Outcome,
		"billing_product", committed.Product,
		"market_code", committed.MarketCode,
		"currency", committed.Currency,
		"tier", committed.Tier,
		"unit_cost_units", committed.UnitCostUnits,
		"quantity", committed.Quantity,
		"amount_units", committed.AmountUnits,
		"full_cost_units", committed.FullCostUnits,
		"credit_consumed_units", committed.CreditConsumedUnits,
		"wallet_debit_units", committed.WalletDebitUnits,
		"remaining_credit_units", committed.RemainingCreditUnits,
		"remaining_balance_units", committed.RemainingBalance,
	)
}
