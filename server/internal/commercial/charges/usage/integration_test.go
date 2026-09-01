package usage

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCommunicationCreditUsageIntegration(t *testing.T) {
	pool := usageTestPool(t)
	service := NewService(NewRepository(pool))

	t.Run("mixed email and SMS share one credit balance", func(t *testing.T) {
		fixture := seedUsageCredit(t, pool, 349_000_000, 100_000_000)
		seedScaleRates(t, pool)

		email := processUsageEmail(t, pool, service, EmailChargeInput{
			TeamID: fixture.teamID, MessageID: uuid.New(), RecipientCount: 10_000,
		})
		sms := processUsageSMS(t, pool, service, SMSChargeInput{
			TeamID: fixture.teamID, MessageID: uuid.New(), DestinationNumber: "+233201234567", Segments: 2_000,
		})

		if email.FullCostUnits != 70_440_000 || email.CreditConsumedUnits != 70_440_000 || email.WalletDebitUnits != 0 {
			t.Fatalf("email charge = %+v", email)
		}
		if sms.FullCostUnits != 110_000_000 || sms.CreditConsumedUnits != 110_000_000 || sms.WalletDebitUnits != 0 {
			t.Fatalf("SMS charge = %+v", sms)
		}
		assertUsageCredit(t, pool, fixture, 180_440_000, 168_560_000, 100_000_000)
	})

	t.Run("partial credit falls through to wallet overage", func(t *testing.T) {
		fixture := seedUsageCredit(t, pool, 10_000_000, 20_000_000)
		seedScaleRates(t, pool)

		charge := processUsageSMS(t, pool, service, SMSChargeInput{
			TeamID: fixture.teamID, MessageID: uuid.New(), DestinationNumber: "+233201234567", Segments: 200,
		})
		if charge.FullCostUnits != 11_000_000 || charge.CreditConsumedUnits != 10_000_000 || charge.WalletDebitUnits != 1_000_000 || charge.RemainingCreditUnits != 0 {
			t.Fatalf("charge = %+v", charge)
		}
		assertUsageCredit(t, pool, fixture, 10_000_000, 0, 19_000_000)
	})

	t.Run("international SMS consumes final wallet-currency cost", func(t *testing.T) {
		fixture := seedUsageCredit(t, pool, 20_000_000, 20_000_000)
		seedScaleRates(t, pool)

		charge := processUsageSMS(t, pool, service, SMSChargeInput{
			TeamID: fixture.teamID, MessageID: uuid.New(), DestinationNumber: "+254712345678", Segments: 100,
		})
		if charge.UnitCostUnits != 176_100 || charge.FullCostUnits != 17_610_000 {
			t.Fatalf("international pricing = %+v", charge)
		}
		if charge.CreditConsumedUnits != 17_610_000 || charge.WalletDebitUnits != 0 || charge.RemainingCreditUnits != 2_390_000 {
			t.Fatalf("international charge = %+v", charge)
		}
		assertUsageCredit(t, pool, fixture, 17_610_000, 2_390_000, 20_000_000)
	})

	t.Run("concurrent channels cannot overspend communication credit", func(t *testing.T) {
		fixture := seedUsageCredit(t, pool, 10_000_000, 20_000_000)
		seedScaleRates(t, pool)

		type result struct {
			charge Charge
			err    error
		}
		results := make(chan result, 2)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			charge, err := processUsageEmailConcurrent(pool, service, EmailChargeInput{
				TeamID: fixture.teamID, MessageID: uuid.New(), RecipientCount: 1_000,
			})
			results <- result{charge: charge, err: err}
		}()
		go func() {
			defer wg.Done()
			charge, err := processUsageSMSConcurrent(pool, service, SMSChargeInput{
				TeamID: fixture.teamID, MessageID: uuid.New(), DestinationNumber: "+233201234567", Segments: 100,
			})
			results <- result{charge: charge, err: err}
		}()
		wg.Wait()
		close(results)

		var fullCost, creditConsumed, walletDebit int64
		for item := range results {
			if item.err != nil {
				t.Fatal(item.err)
			}
			fullCost += item.charge.FullCostUnits
			creditConsumed += item.charge.CreditConsumedUnits
			walletDebit += item.charge.WalletDebitUnits
		}
		if fullCost != 12_544_000 || creditConsumed != 10_000_000 || walletDebit != 2_544_000 {
			t.Fatalf("aggregate full=%d credit=%d wallet=%d", fullCost, creditConsumed, walletDebit)
		}
		assertUsageCredit(t, pool, fixture, 10_000_000, 0, 17_456_000)

		var authorizations int
		if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM usage_authorizations WHERE team_id=$1`, fixture.teamID).Scan(&authorizations); err != nil {
			t.Fatal(err)
		}
		if authorizations != 2 {
			t.Fatalf("usage authorizations = %d, want 2", authorizations)
		}
	})
}

type usageFixture struct {
	teamID   uuid.UUID
	creditID uuid.UUID
}

func seedUsageCredit(t *testing.T, pool *pgxpool.Pool, grantedUnits, walletBalance int64) usageFixture {
	t.Helper()
	ctx := context.Background()
	teamID := uuid.New()
	subscriptionID := uuid.New()
	creditID := uuid.New()
	periodStart := time.Now().UTC().Add(-time.Hour)
	periodEnd := periodStart.AddDate(0, 1, 0)

	batch := &pgx.Batch{}
	batch.Queue(`INSERT INTO teams(id,status,market_code) VALUES($1,'active','GH')`, teamID)
	batch.Queue(`INSERT INTO team_wallets(team_id,billing_market,currency,balance_units) VALUES($1,'GH','GHS',$2)`, teamID, walletBalance)
	batch.Queue(`INSERT INTO team_subscriptions(id,team_id,plan_code,status,current_period_start,current_period_end) VALUES($1,$2,'scale','active',$3,$4)`, subscriptionID, teamID, periodStart, periodEnd)
	batch.Queue(`INSERT INTO subscription_credits(id,subscription_id,team_id,currency,period_start,period_end,granted_units,consumed_units) VALUES($1,$2,$3,'GHS',$4,$5,$6,0)`, creditID, subscriptionID, teamID, periodStart, periodEnd, grantedUnits)
	results := pool.SendBatch(ctx, batch)
	defer func() {
		if err := results.Close(); err != nil {
			t.Errorf("close batch results: %v", err)
		}
	}()
	for range 4 {
		if _, err := results.Exec(); err != nil {
			t.Fatal(err)
		}
	}

	return usageFixture{teamID: teamID, creditID: creditID}
}

func seedScaleRates(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO product_rates(id,product,meter,billing_market,tier,currency,cost_units,effective_from)
		VALUES(gen_random_uuid(),'email','email_recipient','GH','scale','GHS',7044,now()-interval '1 day');
		INSERT INTO sms_rates(id,destination_country,route_type,tier,currency,cost_units,effective_from)
		VALUES(gen_random_uuid(),'GH','local','scale','GHS',55000,now()-interval '1 day'),
		      (gen_random_uuid(),'KE','intl','scale','USD',15000,now()-interval '1 day');
		INSERT INTO fx_rates(id,base_currency,quote_currency,rate,effective_from)
		VALUES(gen_random_uuid(),'USD','GHS',11.74,now()-interval '1 day')`); err != nil {
		t.Fatal(err)
	}
}

func processUsageEmail(t *testing.T, pool *pgxpool.Pool, service *Service, input EmailChargeInput) Charge {
	t.Helper()
	charge, err := processUsageEmailConcurrent(pool, service, input)
	if err != nil {
		t.Fatal(err)
	}
	return charge
}

func processUsageSMS(t *testing.T, pool *pgxpool.Pool, service *Service, input SMSChargeInput) Charge {
	t.Helper()
	charge, err := processUsageSMSConcurrent(pool, service, input)
	if err != nil {
		t.Fatal(err)
	}
	return charge
}

func processUsageEmailConcurrent(pool *pgxpool.Pool, service *Service, input EmailChargeInput) (Charge, error) {
	ctx := context.Background()
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Charge{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	charge, err := service.ChargeEmail(ctx, tx, input)
	if err != nil {
		return Charge{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Charge{}, err
	}
	return charge, nil
}

func processUsageSMSConcurrent(pool *pgxpool.Pool, service *Service, input SMSChargeInput) (Charge, error) {
	ctx := context.Background()
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Charge{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	charge, err := service.ChargeSMS(ctx, tx, input)
	if err != nil {
		return Charge{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Charge{}, err
	}
	return charge, nil
}

func assertUsageCredit(t *testing.T, pool *pgxpool.Pool, fixture usageFixture, consumedUnits, remainingUnits, walletBalance int64) {
	t.Helper()
	var consumed, remaining, balance int64
	err := pool.QueryRow(context.Background(), `
		SELECT credit.consumed_units,
		       GREATEST(credit.granted_units-credit.consumed_units,0),
		       wallet.balance_units
		FROM subscription_credits credit
		JOIN team_wallets wallet ON wallet.team_id=credit.team_id
		WHERE credit.id=$1`, fixture.creditID).Scan(&consumed, &remaining, &balance)
	if err != nil {
		t.Fatal(err)
	}
	if consumed != consumedUnits || remaining != remainingUnits || balance != walletBalance {
		t.Fatalf("credit consumed=%d remaining=%d wallet=%d; want %d/%d/%d", consumed, remaining, balance, consumedUnits, remainingUnits, walletBalance)
	}
	if consumed < 0 || remaining < 0 {
		t.Fatalf("negative communication credit state consumed=%d remaining=%d", consumed, remaining)
	}
}

func usageTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	schema := "usage_credit_" + uuid.New().String()[:8]
	admin, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), `DROP SCHEMA `+schema+` CASCADE`)
		admin.Close()
	})
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, usageCreditTestSchema); err != nil {
		t.Fatal(fmt.Errorf("create usage credit test schema: %w", err))
	}
	return pool
}

const usageCreditTestSchema = `
CREATE TABLE teams(
    id uuid PRIMARY KEY,
    status text NOT NULL,
    market_code char(2) NOT NULL
);
CREATE TABLE billing_markets(
    code char(2) PRIMARY KEY,
    currency char(3) NOT NULL,
    is_enabled boolean NOT NULL DEFAULT true
);
INSERT INTO billing_markets(code,currency) VALUES('GH','GHS');
CREATE TABLE team_wallets(
    team_id uuid PRIMARY KEY REFERENCES teams(id),
    billing_market char(2) NOT NULL,
    currency char(3) NOT NULL,
    balance_units bigint NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE team_subscriptions(
    id uuid PRIMARY KEY,
    team_id uuid NOT NULL UNIQUE REFERENCES teams(id),
    plan_code text NOT NULL,
    status text NOT NULL,
    current_period_start timestamptz NOT NULL,
    current_period_end timestamptz NOT NULL
);
CREATE TABLE subscription_credits(
    id uuid PRIMARY KEY,
    subscription_id uuid NOT NULL,
    subscription_charge_id uuid,
    team_id uuid NOT NULL,
    plan_code text NOT NULL DEFAULT 'scale',
    billing_market char(2) NOT NULL DEFAULT 'GH',
    currency char(3) NOT NULL,
    period_start timestamptz NOT NULL,
    period_end timestamptz NOT NULL,
    granted_units bigint NOT NULL,
    consumed_units bigint NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE allowance_policies(
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    product text NOT NULL,
    meter text NOT NULL,
    billing_market char(2) NOT NULL,
    tier text NOT NULL,
    included_quantity bigint NOT NULL,
    cadence text NOT NULL DEFAULT 'monthly',
    effective_from timestamptz NOT NULL,
    effective_until timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE usage_allowances(
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id uuid NOT NULL,
    allowance_policy_id uuid NOT NULL,
    product text NOT NULL,
    meter text NOT NULL,
    billing_market char(2) NOT NULL,
    tier text NOT NULL,
    period_start timestamptz NOT NULL,
    period_end timestamptz NOT NULL,
    included_quantity bigint NOT NULL,
    consumed_quantity bigint NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE(team_id,product,meter,period_start,period_end)
);
CREATE TABLE product_rates(
    id uuid PRIMARY KEY,
    product text NOT NULL,
    meter text NOT NULL,
    billing_market char(2) NOT NULL,
    tier text NOT NULL,
    currency char(3) NOT NULL,
    cost_units bigint NOT NULL,
    effective_from timestamptz NOT NULL,
    effective_until timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE sms_rates(
    id uuid PRIMARY KEY,
    destination_country char(2) NOT NULL,
    route_type text NOT NULL,
    tier text NOT NULL,
    currency char(3) NOT NULL,
    cost_units bigint NOT NULL,
    effective_from timestamptz NOT NULL,
    effective_until timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE fx_rates(
    id uuid PRIMARY KEY,
    base_currency char(3) NOT NULL,
    quote_currency char(3) NOT NULL,
    rate numeric NOT NULL,
    effective_from timestamptz NOT NULL,
    effective_until timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE usage_authorizations(
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id uuid NOT NULL,
    product text NOT NULL,
    meter text NOT NULL,
    reference_id text NOT NULL,
    usage_allowance_id uuid,
    sms_rate_id uuid,
    fx_rate_id uuid,
    product_rate_id uuid,
    billing_market char(2) NOT NULL,
    destination_country char(2),
    route_type text,
    total_quantity bigint NOT NULL,
    allowance_quantity bigint NOT NULL DEFAULT 0,
    billable_quantity bigint NOT NULL DEFAULT 0,
    unit_cost_units bigint NOT NULL DEFAULT 0,
    amount_units bigint NOT NULL DEFAULT 0,
    subscription_credit_id uuid,
    full_cost_units bigint NOT NULL DEFAULT 0,
    credit_consumed_units bigint NOT NULL DEFAULT 0,
    wallet_debit_units bigint NOT NULL DEFAULT 0,
    currency char(3) NOT NULL,
    tier text NOT NULL,
    priced_at timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE(team_id,product,meter,reference_id)
);
CREATE TABLE wallet_ledger(
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id uuid NOT NULL REFERENCES teams(id),
    usage_authorization_id uuid,
    amount_units bigint NOT NULL,
    transaction_type text NOT NULL,
    reference_id text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE(team_id,transaction_type,reference_id)
);
`
