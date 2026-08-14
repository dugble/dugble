package usage

import "testing"

func TestBalanceLevel(t *testing.T) {
	tests := []struct {
		name   string
		charge CommittedCharge
		want   string
	}{
		{name: "exhausted", charge: CommittedCharge{Charge: Charge{RemainingBalance: 0, WalletDebitUnits: 100}}, want: BalanceLevelExhausted},
		{name: "low", charge: CommittedCharge{Charge: Charge{RemainingBalance: 99, WalletDebitUnits: 100}}, want: BalanceLevelLow},
		{name: "enough for similar charge", charge: CommittedCharge{Charge: Charge{RemainingBalance: 100, WalletDebitUnits: 100}}},
		{name: "credit-only charge", charge: CommittedCharge{Charge: Charge{RemainingBalance: 1, WalletDebitUnits: 0}}},
		{name: "credit-only charge with empty wallet", charge: CommittedCharge{Charge: Charge{RemainingBalance: 0, WalletDebitUnits: 0}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := balanceLevel(test.charge); got != test.want {
				t.Fatalf("balanceLevel() = %q, want %q", got, test.want)
			}
		})
	}
}
