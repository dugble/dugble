package usage

import (
	"context"
	"fmt"

	sentrymonitoring "github.com/dugble/dugble/server/internal/integrations/monitoring/sentry"
	"github.com/dugble/dugble/server/internal/messaging/email/systemmail"
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
	if s.balanceNotifier != nil && s.recipients != nil {
		if err := s.notifyBalance(ctx, committed); err != nil {
			sentrymonitoring.Warn("failed to send wallet balance notification", "team_id", committed.TeamID, "error", err)
		}
	}
}

func (s *Service) notifyBalance(ctx context.Context, charge CommittedCharge) error {
	level := balanceLevel(charge)
	if level == "" {
		return nil
	}
	recipients, err := s.recipients.ListBalanceRecipients(ctx, charge.TeamID)
	if err != nil {
		return err
	}
	for _, recipient := range recipients {
		input := systemmail.SendWalletBalanceAlertInput{
			ToEmail: recipient.Email, Name: recipient.Name, TeamName: recipient.TeamName,
			Currency: charge.Currency, BalanceUnits: charge.RemainingBalance, Level: level,
		}
		if err := s.balanceNotifier.SendWalletBalanceAlert(ctx, input); err != nil {
			return fmt.Errorf("send wallet balance notification: %w", err)
		}
	}
	return nil
}

func balanceLevel(charge CommittedCharge) string {
	if charge.WalletDebitUnits > 0 && charge.RemainingBalance <= 0 {
		return "exhausted"
	}
	if charge.WalletDebitUnits > 0 && charge.RemainingBalance < charge.WalletDebitUnits {
		return "low"
	}
	return ""
}
