package usage

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestGrowthEmailAllowanceIntegration(t *testing.T) {
	t.Run("free usage consumes allowance before billing", func(t *testing.T) {
		pool := usageTestPool(t)
		service := NewService(NewRepository(pool))
		teamID := seedGrowthEmailAllowance(t, pool, 10_000_000)

		charge := processUsageEmail(t, pool, service, EmailChargeInput{
			TeamID: teamID, MessageID: uuid.New(), RecipientCount: 100,
		})
		if charge.Outcome != OutcomeApplied || charge.FullCostUnits != 0 || charge.WalletDebitUnits != 0 || charge.UnitCostUnits != 0 {
			t.Fatalf("free allowance charge = %+v", charge)
		}
		assertEmailAllowanceUsage(t, pool, teamID, 100, 0, 100)
	})

	t.Run("crossing allowance boundary bills only overage", func(t *testing.T) {
		pool := usageTestPool(t)
		service := NewService(NewRepository(pool))
		teamID := seedGrowthEmailAllowance(t, pool, 10_000_000)

		first := processUsageEmail(t, pool, service, EmailChargeInput{
			TeamID: teamID, MessageID: uuid.New(), RecipientCount: 995,
		})
		if first.FullCostUnits != 0 || first.WalletDebitUnits != 0 {
			t.Fatalf("first charge = %+v", first)
		}

		secondID := uuid.New()
		second := processUsageEmail(t, pool, service, EmailChargeInput{
			TeamID: teamID, MessageID: secondID, RecipientCount: 10,
		})
		const wantOverageCost int64 = 5 * 9392
		if second.UnitCostUnits != 9392 || second.FullCostUnits != wantOverageCost || second.WalletDebitUnits != wantOverageCost {
			t.Fatalf("boundary charge = %+v", second)
		}

		var allowanceQuantity, billableQuantity int64
		if err := pool.QueryRow(context.Background(), `
			SELECT allowance_quantity, billable_quantity
			FROM usage_authorizations
			WHERE team_id=$1 AND reference_id=$2`, teamID, secondID.String()).Scan(&allowanceQuantity, &billableQuantity); err != nil {
			t.Fatal(err)
		}
		if allowanceQuantity != 5 || billableQuantity != 5 {
			t.Fatalf("boundary quantities allowance=%d billable=%d, want 5/5", allowanceQuantity, billableQuantity)
		}
		assertEmailAllowanceConsumed(t, pool, teamID, 1000)
	})

	t.Run("duplicate charge does not consume allowance twice", func(t *testing.T) {
		pool := usageTestPool(t)
		service := NewService(NewRepository(pool))
		teamID := seedGrowthEmailAllowance(t, pool, 10_000_000)
		messageID := uuid.New()
		input := EmailChargeInput{TeamID: teamID, MessageID: messageID, RecipientCount: 250}

		first := processUsageEmail(t, pool, service, input)
		second := processUsageEmail(t, pool, service, input)
		if first.Outcome != OutcomeApplied || second.Outcome != OutcomeAlreadyApplied {
			t.Fatalf("idempotent outcomes first=%s second=%s", first.Outcome, second.Outcome)
		}
		assertEmailAllowanceConsumed(t, pool, teamID, 250)

		var authorizations int
		if err := pool.QueryRow(context.Background(), `
			SELECT count(*) FROM usage_authorizations
			WHERE team_id=$1 AND product='email'`, teamID).Scan(&authorizations); err != nil {
			t.Fatal(err)
		}
		if authorizations != 1 {
			t.Fatalf("usage authorizations = %d, want 1", authorizations)
		}
	})

	t.Run("concurrent sends cannot overspend allowance", func(t *testing.T) {
		pool := usageTestPool(t)
		service := NewService(NewRepository(pool))
		teamID := seedGrowthEmailAllowance(t, pool, 10_000_000)

		type result struct {
			charge Charge
			err    error
		}
		results := make(chan result, 2)
		var wg sync.WaitGroup
		wg.Add(2)
		for range 2 {
			go func() {
				defer wg.Done()
				charge, err := processUsageEmailConcurrent(pool, service, EmailChargeInput{
					TeamID: teamID, MessageID: uuid.New(), RecipientCount: 600,
				})
				results <- result{charge: charge, err: err}
			}()
		}
		wg.Wait()
		close(results)

		var fullCost, walletDebit int64
		for item := range results {
			if item.err != nil {
				t.Fatal(item.err)
			}
			fullCost += item.charge.FullCostUnits
			walletDebit += item.charge.WalletDebitUnits
		}
		const wantOverageCost int64 = 200 * 9392
		if fullCost != wantOverageCost || walletDebit != wantOverageCost {
			t.Fatalf("aggregate full=%d wallet=%d, want %d", fullCost, walletDebit, wantOverageCost)
		}

		var allowanceQuantity, billableQuantity int64
		if err := pool.QueryRow(context.Background(), `
			SELECT COALESCE(sum(allowance_quantity),0), COALESCE(sum(billable_quantity),0)
			FROM usage_authorizations
			WHERE team_id=$1 AND product='email'`, teamID).Scan(&allowanceQuantity, &billableQuantity); err != nil {
			t.Fatal(err)
		}
		if allowanceQuantity != 1000 || billableQuantity != 200 {
			t.Fatalf("aggregate quantities allowance=%d billable=%d, want 1000/200", allowanceQuantity, billableQuantity)
		}
		assertEmailAllowanceConsumed(t, pool, teamID, 1000)
	})
}

func seedGrowthEmailAllowance(t *testing.T, pool *pgxpool.Pool, walletBalance int64) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	teamID := uuid.New()
	subscriptionID := uuid.New()
	now := time.Now().UTC()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, 0)

	if _, err := pool.Exec(ctx, `
		INSERT INTO teams(id,status,market_code)
		VALUES($1,'active','GH')`, teamID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO team_wallets(team_id,billing_market,currency,balance_units)
		VALUES($1,'GH','GHS',$2)`, teamID, walletBalance); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO team_subscriptions(id,team_id,plan_code,status,current_period_start,current_period_end)
		VALUES($1,$2,'growth','active',$3,$4)`, subscriptionID, teamID, periodStart, periodEnd); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO product_rates(id,product,meter,billing_market,tier,currency,cost_units,effective_from)
		VALUES(gen_random_uuid(),'email','email_recipient','GH','growth','GHS',9392,$1)`, periodStart); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO allowance_policies(id,product,meter,billing_market,tier,included_quantity,cadence,effective_from)
		VALUES(gen_random_uuid(),'email','email_recipient','GH','growth',1000,'monthly',$1)`, periodStart); err != nil {
		t.Fatal(err)
	}
	return teamID
}

func assertEmailAllowanceUsage(t *testing.T, pool *pgxpool.Pool, teamID uuid.UUID, wantAllowance, wantBillable, wantConsumed int64) {
	t.Helper()
	var allowanceQuantity, billableQuantity int64
	if err := pool.QueryRow(context.Background(), `
		SELECT allowance_quantity, billable_quantity
		FROM usage_authorizations
		WHERE team_id=$1 AND product='email'`, teamID).Scan(&allowanceQuantity, &billableQuantity); err != nil {
		t.Fatal(err)
	}
	if allowanceQuantity != wantAllowance || billableQuantity != wantBillable {
		t.Fatalf("usage quantities allowance=%d billable=%d, want %d/%d", allowanceQuantity, billableQuantity, wantAllowance, wantBillable)
	}
	assertEmailAllowanceConsumed(t, pool, teamID, wantConsumed)
}

func assertEmailAllowanceConsumed(t *testing.T, pool *pgxpool.Pool, teamID uuid.UUID, want int64) {
	t.Helper()
	var included, consumed int64
	if err := pool.QueryRow(context.Background(), `
		SELECT included_quantity, consumed_quantity
		FROM usage_allowances
		WHERE team_id=$1 AND product='email' AND meter='email_recipient'`, teamID).Scan(&included, &consumed); err != nil {
		t.Fatal(err)
	}
	if included != 1000 || consumed != want {
		t.Fatalf("allowance included=%d consumed=%d, want 1000/%d", included, consumed, want)
	}
}
