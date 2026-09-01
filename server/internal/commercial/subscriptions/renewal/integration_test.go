package renewal

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	charges "github.com/dugble/dugble/server/internal/commercial/charges/subscription"
	usagecharges "github.com/dugble/dugble/server/internal/commercial/charges/usage"
	"github.com/dugble/dugble/server/internal/commercial/subscriptions/lifecycle"
	"github.com/dugble/dugble/server/internal/modules/outbox"
)

func TestRenewalIntegration(t *testing.T) {
	pool := renewalTestPool(t)
	service := NewService(NewRepository(), charges.NewService(charges.NewRepository()), lifecycle.NewService()).
		WithEventPublisher(NewEventPublisher(outbox.NewRepository(pool)))

	t.Run("successful renewal debits wallet once", func(t *testing.T) {
		fixture := seedRenewal(t, pool, seedOptions{balance: 500, price: 100})
		result := processRenewal(t, pool, service, fixture.teamID)
		if result.Outcome != OutcomeRenewed || result.Charge.Outcome != charges.OutcomeApplied {
			t.Fatalf("result = %+v", result)
		}
		if result.Charge.CreditID == nil || result.Charge.CreditGranted != 100 {
			t.Fatalf("credit result = %+v", result.Charge)
		}
		assertRenewalState(t, pool, fixture, "active", "growth", 400, 1, 1)
		assertSubscriptionCredit(t, pool, fixture.teamID, 1, 100)
		assertRenewalEvent(t, pool, fixture.teamID, "billing.subscription.renewed", 1)
		second := processRenewal(t, pool, service, fixture.teamID)
		if second.Outcome != OutcomeNotDue {
			t.Fatalf("second outcome = %q", second.Outcome)
		}
		assertRenewalState(t, pool, fixture, "active", "growth", 400, 1, 1)
	})

	t.Run("plan change charges and activates next plan", func(t *testing.T) {
		fixture := seedRenewal(t, pool, seedOptions{balance: 500_000_000, price: 349_000_000, pendingPlan: "scale"})
		result := processRenewal(t, pool, service, fixture.teamID)
		if result.Outcome != OutcomePlanChanged || result.CurrentPlan != "scale" {
			t.Fatalf("result = %+v", result)
		}
		assertRenewalState(t, pool, fixture, "active", "scale", 151_000_000, 1, 1)
		assertSubscriptionCredit(t, pool, fixture.teamID, 1, 349_000_000)

		if _, err := pool.Exec(context.Background(), `INSERT INTO product_rates(id,product,meter,billing_market,tier,currency,cost_units,effective_from) VALUES(gen_random_uuid(),'email','email_recipient','GH','scale','GHS',7044,'2026-08-01')`); err != nil {
			t.Fatal(err)
		}
		emailCharge := processEmailCharge(t, pool, usagecharges.NewService(usagecharges.NewRepository(pool)), usagecharges.EmailChargeInput{
			TeamID: fixture.teamID, MessageID: uuid.New(), RecipientCount: 1_000,
		})
		if emailCharge.FullCostUnits != 7_044_000 || emailCharge.CreditConsumedUnits != 7_044_000 || emailCharge.WalletDebitUnits != 0 {
			t.Fatalf("email charge = %+v", emailCharge)
		}
		assertSubscriptionCredit(t, pool, fixture.teamID, 1, 349_000_000-7_044_000)
	})

	t.Run("failed pending plan remains inactive and grants no credit", func(t *testing.T) {
		fixture := seedRenewal(t, pool, seedOptions{balance: 100_000_000, price: 349_000_000, pendingPlan: "scale"})
		result := processRenewal(t, pool, service, fixture.teamID)
		if result.Outcome != OutcomePastDue || result.CurrentPlan != "growth" {
			t.Fatalf("result = %+v", result)
		}
		assertRenewalState(t, pool, fixture, "past_due", "growth", 100_000_000, 1, 0)
		assertSubscriptionCredit(t, pool, fixture.teamID, 0, 0)
	})

	t.Run("SMS consumes final wallet-currency cost from credit", func(t *testing.T) {
		fixture := seedRenewal(t, pool, seedOptions{balance: 500_000_000, price: 349_000_000, pendingPlan: "scale"})
		result := processRenewal(t, pool, service, fixture.teamID)
		if result.Outcome != OutcomePlanChanged {
			t.Fatalf("renewal = %+v", result)
		}
		if _, err := pool.Exec(context.Background(), `
			INSERT INTO sms_rates(id,destination_country,route_type,tier,currency,cost_units,effective_from)
			VALUES(gen_random_uuid(),'GH','local','scale','GHS',55000,'2026-08-01'),
			      (gen_random_uuid(),'NG','intl','scale','USD',15000,'2026-08-01');
			INSERT INTO fx_rates(id,base_currency,quote_currency,rate,effective_from)
			VALUES(gen_random_uuid(),'USD','GHS',11.74,'2026-08-01')`); err != nil {
			t.Fatal(err)
		}
		charger := usagecharges.NewService(usagecharges.NewRepository(pool))
		local := processSMSCharge(t, pool, charger, usagecharges.SMSChargeInput{
			TeamID: fixture.teamID, MessageID: uuid.New(), DestinationNumber: "+233201234567", Segments: 3_000,
		})
		if local.FullCostUnits != 165_000_000 || local.CreditConsumedUnits != 165_000_000 || local.WalletDebitUnits != 0 {
			t.Fatalf("local SMS = %+v", local)
		}
		international := processSMSCharge(t, pool, charger, usagecharges.SMSChargeInput{
			TeamID: fixture.teamID, MessageID: uuid.New(), DestinationNumber: "+2348012345678", Segments: 100,
		})
		if international.UnitCostUnits != 176_100 || international.FullCostUnits != 17_610_000 || international.CreditConsumedUnits != 17_610_000 || international.WalletDebitUnits != 0 {
			t.Fatalf("international SMS = %+v", international)
		}
		assertSubscriptionCredit(t, pool, fixture.teamID, 1, 166_390_000)
	})

	t.Run("insufficient balance is retryable after funding", func(t *testing.T) {
		fixture := seedRenewal(t, pool, seedOptions{balance: 50, price: 100})
		failed := processRenewal(t, pool, service, fixture.teamID)
		if failed.Outcome != OutcomePastDue || failed.Charge.AttemptCount != 1 {
			t.Fatalf("failed result = %+v", failed)
		}
		assertRenewalState(t, pool, fixture, "past_due", "growth", 50, 1, 0)
		assertSubscriptionCredit(t, pool, fixture.teamID, 0, 0)
		if _, err := pool.Exec(context.Background(), `UPDATE team_wallets SET balance_units = 150 WHERE team_id = $1`, fixture.teamID); err != nil {
			t.Fatal(err)
		}
		retried := processRenewal(t, pool, service, fixture.teamID)
		if retried.Outcome != OutcomeRenewed || retried.Charge.AttemptCount != 2 || retried.Charge.AppliedAt == nil {
			t.Fatalf("retried result = %+v", retried)
		}
		assertRenewalState(t, pool, fixture, "active", "growth", 50, 1, 1)
		assertSubscriptionCredit(t, pool, fixture.teamID, 1, 100)
	})

	t.Run("cancellation creates no charge and preserves final period", func(t *testing.T) {
		fixture := seedRenewal(t, pool, seedOptions{balance: 500, price: 100, cancel: true})
		result := processRenewal(t, pool, service, fixture.teamID)
		if result.Outcome != OutcomeCanceled || !result.PeriodStart.Equal(fixture.periodStart) || !result.PeriodEnd.Equal(fixture.periodEnd) {
			t.Fatalf("result = %+v", result)
		}
		assertRenewalState(t, pool, fixture, "canceled", "growth", 500, 0, 0)
	})

	t.Run("zero cost plan advances without ledger debit", func(t *testing.T) {
		fixture := seedRenewal(t, pool, seedOptions{balance: 25, price: 0})
		result := processRenewal(t, pool, service, fixture.teamID)
		if result.Outcome != OutcomeRenewed || result.Charge.Outcome != charges.OutcomeApplied {
			t.Fatalf("result = %+v", result)
		}
		assertRenewalState(t, pool, fixture, "active", "growth", 25, 1, 0)
		assertSubscriptionCredit(t, pool, fixture.teamID, 0, 0)
	})

	t.Run("commitment charge retry returns one credit grant", func(t *testing.T) {
		fixture := seedRenewal(t, pool, seedOptions{balance: 500_000_000, price: 349_000_000})
		charger := charges.NewService(charges.NewRepository())
		input := charges.Input{
			SubscriptionID: fixture.subscriptionID,
			TeamID:         fixture.teamID,
			PlanCode:       "growth",
			PeriodStart:    fixture.periodEnd,
			PeriodEnd:      fixture.periodEnd.AddDate(0, 1, 0),
		}
		first := processCommitmentCharge(t, pool, charger, input)
		second := processCommitmentCharge(t, pool, charger, input)
		if first.Outcome != charges.OutcomeApplied || second.Outcome != charges.OutcomeAlreadyApplied {
			t.Fatalf("first=%+v second=%+v", first, second)
		}
		if first.CreditID == nil || second.CreditID == nil || *first.CreditID != *second.CreditID {
			t.Fatalf("first credit=%v second credit=%v", first.CreditID, second.CreditID)
		}
		if first.CreditGranted != 349_000_000 || second.CreditGranted != 349_000_000 {
			t.Fatalf("first grant=%d second grant=%d", first.CreditGranted, second.CreditGranted)
		}
		assertRenewalState(t, pool, fixture, "active", "growth", 151_000_000, 1, 1)
		assertSubscriptionCredit(t, pool, fixture.teamID, 1, 349_000_000)
	})

	t.Run("unused credit expires and next period receives a fresh grant", func(t *testing.T) {
		fixture := seedRenewal(t, pool, seedOptions{balance: 800_000_000, price: 349_000_000})
		charger := charges.NewService(charges.NewRepository())
		old := processCommitmentCharge(t, pool, charger, charges.Input{
			SubscriptionID: fixture.subscriptionID, TeamID: fixture.teamID, PlanCode: "growth",
			PeriodStart: fixture.periodStart, PeriodEnd: fixture.periodEnd,
		})
		if old.CreditID == nil {
			t.Fatalf("old charge = %+v", old)
		}
		if _, err := pool.Exec(context.Background(), `UPDATE subscription_credits SET consumed_units = 240000000 WHERE id = $1`, *old.CreditID); err != nil {
			t.Fatal(err)
		}
		result := processRenewal(t, pool, service, fixture.teamID)
		if result.Outcome != OutcomeRenewed || result.Charge.CreditID == nil {
			t.Fatalf("renewal = %+v", result)
		}
		var oldRemaining, newRemaining int64
		err := pool.QueryRow(context.Background(), `SELECT
			max(granted_units-consumed_units) FILTER (WHERE period_start=$2),
			max(granted_units-consumed_units) FILTER (WHERE period_start=$3)
			FROM subscription_credits WHERE team_id=$1`, fixture.teamID, fixture.periodStart, fixture.periodEnd).Scan(&oldRemaining, &newRemaining)
		if err != nil {
			t.Fatal(err)
		}
		if oldRemaining != 109_000_000 || newRemaining != 349_000_000 {
			t.Fatalf("old remaining=%d new remaining=%d", oldRemaining, newRemaining)
		}
	})
}

func TestRenewalWorkerContinuesAfterTeamFailure(t *testing.T) {
	pool := renewalTestPool(t)
	service := NewService(NewRepository(), charges.NewService(charges.NewRepository()), lifecycle.NewService()).
		WithEventPublisher(NewEventPublisher(outbox.NewRepository(pool)))
	broken := seedRenewal(t, pool, seedOptions{balance: 500, price: 100})
	healthy := seedRenewal(t, pool, seedOptions{balance: 500, price: 100})
	wantErr := errors.New("broken team")
	processor := processorFunc(func(ctx context.Context, tx pgx.Tx, teamID uuid.UUID) (Result, error) {
		if teamID == broken.teamID {
			return Result{}, wantErr
		}
		return service.ProcessTeam(ctx, tx, teamID)
	})
	var observed Failure
	worker, err := NewWorker(pool, processor, Config{
		PollInterval: time.Minute,
		BatchSize:    10,
		OnFailure: func(_ context.Context, failure Failure) {
			observed = failure
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := worker.ProcessBatch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Processed != 2 || result.Renewed != 1 || len(result.Failures) != 1 {
		t.Fatalf("result = %+v", result)
	}
	if result.Failures[0].TeamID != broken.teamID || !errors.Is(result.Failures[0].Err, wantErr) {
		t.Fatalf("failure = %+v", result.Failures[0])
	}
	if observed.TeamID != broken.teamID || !errors.Is(observed.Err, wantErr) {
		t.Fatalf("observed failure = %+v", observed)
	}
	assertRenewalState(t, pool, broken, "active", "growth", 500, 0, 0)
	assertRenewalState(t, pool, healthy, "active", "growth", 400, 1, 1)
	assertSubscriptionCredit(t, pool, broken.teamID, 0, 0)
	assertSubscriptionCredit(t, pool, healthy.teamID, 1, 100)
	assertRenewalEvent(t, pool, broken.teamID, "", 0)
	assertRenewalEvent(t, pool, healthy.teamID, "billing.subscription.renewed", 1)
}

type processorFunc func(context.Context, pgx.Tx, uuid.UUID) (Result, error)

func (fn processorFunc) ProcessTeam(ctx context.Context, tx pgx.Tx, teamID uuid.UUID) (Result, error) {
	return fn(ctx, tx, teamID)
}

type seedOptions struct {
	balance     int64
	price       int64
	pendingPlan string
	cancel      bool
}

type renewalFixture struct {
	teamID         uuid.UUID
	subscriptionID uuid.UUID
	periodStart    time.Time
	periodEnd      time.Time
}

func seedRenewal(t *testing.T, pool *pgxpool.Pool, options seedOptions) renewalFixture {
	t.Helper()
	ctx := context.Background()
	teamID := uuid.New()
	periodEnd := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	periodStart := periodEnd.AddDate(0, -1, 0)
	for _, plan := range []string{"growth", "scale"} {
		if _, err := pool.Exec(ctx, `INSERT INTO plans(code) VALUES ($1) ON CONFLICT DO NOTHING`, plan); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO teams(id,status,market_code) VALUES ($1,'active','GH')`, teamID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO team_wallets(team_id,billing_market,currency,balance_units) VALUES ($1,'GH','GHS',$2)`, teamID, options.balance); err != nil {
		t.Fatal(err)
	}
	var pending any
	var pendingAt any
	if options.pendingPlan != "" {
		pending, pendingAt = options.pendingPlan, periodEnd
	}
	var subscriptionID uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO team_subscriptions(team_id,plan_code,status,current_period_start,current_period_end,pending_plan_code,pending_plan_effective_at,cancel_at_period_end) VALUES($1,'growth','active',$2,$3,$4,$5,$6) RETURNING id`, teamID, periodStart, periodEnd, pending, pendingAt, options.cancel).Scan(&subscriptionID); err != nil {
		t.Fatal(err)
	}
	pricePlan := "growth"
	if options.pendingPlan != "" {
		pricePlan = options.pendingPlan
	}
	if _, err := pool.Exec(ctx, `UPDATE plan_prices SET effective_until=$2 WHERE plan_code=$1 AND billing_market='GH' AND effective_until IS NULL`, pricePlan, periodStart); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO plan_prices(plan_code,billing_market,currency,amount_units,effective_from) VALUES($1,'GH','GHS',$2,$3)`, pricePlan, options.price, periodStart); err != nil {
		t.Fatal(err)
	}
	return renewalFixture{teamID: teamID, subscriptionID: subscriptionID, periodStart: periodStart, periodEnd: periodEnd}
}

func processCommitmentCharge(t *testing.T, pool *pgxpool.Pool, charger *charges.Service, input charges.Input) charges.Result {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	result, err := charger.ChargePeriod(ctx, tx, input)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return result
}

func processEmailCharge(t *testing.T, pool *pgxpool.Pool, charger *usagecharges.Service, input usagecharges.EmailChargeInput) usagecharges.Charge {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	result, err := charger.ChargeEmail(ctx, tx, input)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return result
}

func processSMSCharge(t *testing.T, pool *pgxpool.Pool, charger *usagecharges.Service, input usagecharges.SMSChargeInput) usagecharges.Charge {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	result, err := charger.ChargeSMS(ctx, tx, input)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return result
}

func processRenewal(t *testing.T, pool *pgxpool.Pool, service *Service, teamID uuid.UUID) Result {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	result, err := service.ProcessTeam(ctx, tx, teamID)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return result
}

func assertRenewalState(t *testing.T, pool *pgxpool.Pool, fixture renewalFixture, status, plan string, balance int64, chargesCount, ledgerCount int) {
	t.Helper()
	ctx := context.Background()
	var gotStatus, gotPlan string
	var gotBalance int64
	var gotCharges, gotLedger int
	err := pool.QueryRow(ctx, `SELECT subscription.status,subscription.plan_code,wallet.balance_units,(SELECT count(*) FROM subscription_charges WHERE team_id=$1),(SELECT count(*) FROM wallet_ledger WHERE team_id=$1 AND transaction_type='subscription') FROM team_subscriptions subscription JOIN team_wallets wallet ON wallet.team_id=subscription.team_id WHERE subscription.team_id=$1`, fixture.teamID).Scan(&gotStatus, &gotPlan, &gotBalance, &gotCharges, &gotLedger)
	if err != nil {
		t.Fatal(err)
	}
	if gotStatus != status || gotPlan != plan || gotBalance != balance || gotCharges != chargesCount || gotLedger != ledgerCount {
		t.Fatalf("state status=%s plan=%s balance=%d charges=%d ledger=%d", gotStatus, gotPlan, gotBalance, gotCharges, gotLedger)
	}
}

func assertRenewalEvent(t *testing.T, pool *pgxpool.Pool, teamID uuid.UUID, subject string, count int) {
	t.Helper()
	var gotCount int
	err := pool.QueryRow(context.Background(), `SELECT count(*) FROM outbox_events WHERE headers->>'team_id' = $1 AND ($2 = '' OR subject = $2)`, teamID.String(), subject).Scan(&gotCount)
	if err != nil {
		t.Fatal(err)
	}
	if gotCount != count {
		t.Fatalf("renewal event count = %d, want %d", gotCount, count)
	}
}

func assertSubscriptionCredit(t *testing.T, pool *pgxpool.Pool, teamID uuid.UUID, count int, remainingUnits int64) {
	t.Helper()
	var gotCount int
	var gotRemaining int64
	err := pool.QueryRow(context.Background(), `SELECT count(*), COALESCE(sum(granted_units - consumed_units), 0)::bigint FROM subscription_credits WHERE team_id = $1`, teamID).Scan(&gotCount, &gotRemaining)
	if err != nil {
		t.Fatal(err)
	}
	if gotCount != count || gotRemaining != remainingUnits {
		t.Fatalf("subscription credits count=%d remaining=%d, want count=%d remaining=%d", gotCount, gotRemaining, count, remainingUnits)
	}
}

func renewalTestPool(t *testing.T) *pgxpool.Pool {
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
	schema := "renewal_" + uuid.New().String()[:8]
	admin, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = admin.Exec(context.Background(), `DROP SCHEMA `+schema+` CASCADE`); admin.Close() })
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, renewalTestSchema); err != nil {
		t.Fatal(fmt.Errorf("create renewal test schema: %w", err))
	}
	return pool
}

const renewalTestSchema = `
	CREATE TABLE teams(id uuid PRIMARY KEY,status text NOT NULL,market_code char(2) NOT NULL);
	CREATE TABLE plans(code text PRIMARY KEY);
	CREATE TABLE billing_markets(code char(2) PRIMARY KEY,currency char(3) NOT NULL,is_enabled boolean NOT NULL DEFAULT true);
	INSERT INTO billing_markets(code,currency) VALUES('GH','GHS');
CREATE TABLE team_wallets(team_id uuid PRIMARY KEY REFERENCES teams(id),billing_market char(2) NOT NULL,currency char(3) NOT NULL,balance_units bigint NOT NULL,created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE plan_prices(id uuid PRIMARY KEY DEFAULT gen_random_uuid(),plan_code text NOT NULL REFERENCES plans(code),billing_market char(2) NOT NULL,currency char(3) NOT NULL,amount_units bigint NOT NULL,billing_interval text NOT NULL DEFAULT 'monthly',effective_from timestamptz NOT NULL,effective_until timestamptz,UNIQUE(id,plan_code,billing_market,currency,amount_units));
CREATE TABLE team_subscriptions(id uuid PRIMARY KEY DEFAULT gen_random_uuid(),team_id uuid NOT NULL UNIQUE REFERENCES teams(id),plan_code text NOT NULL REFERENCES plans(code),status text NOT NULL,current_period_start timestamptz NOT NULL,current_period_end timestamptz NOT NULL,pending_plan_code text REFERENCES plans(code),pending_plan_effective_at timestamptz,cancel_at_period_end boolean NOT NULL DEFAULT false,updated_at timestamptz NOT NULL DEFAULT now(),UNIQUE(id,team_id));
	CREATE TABLE subscription_charges(id uuid PRIMARY KEY DEFAULT gen_random_uuid(),subscription_id uuid NOT NULL,team_id uuid NOT NULL,plan_price_id uuid NOT NULL,plan_code text NOT NULL,billing_market char(2) NOT NULL,currency char(3) NOT NULL,period_start timestamptz NOT NULL,period_end timestamptz NOT NULL,amount_units bigint NOT NULL,status text NOT NULL,failure_code text,attempt_count integer NOT NULL DEFAULT 1,last_attempted_at timestamptz NOT NULL DEFAULT now(),applied_at timestamptz,reference_id text NOT NULL,created_at timestamptz NOT NULL DEFAULT now(),UNIQUE(id,team_id),UNIQUE(subscription_id,period_start),UNIQUE(team_id,reference_id),FOREIGN KEY(subscription_id,team_id) REFERENCES team_subscriptions(id,team_id),FOREIGN KEY(plan_price_id,plan_code,billing_market,currency,amount_units) REFERENCES plan_prices(id,plan_code,billing_market,currency,amount_units));
	CREATE TABLE subscription_credits(id uuid PRIMARY KEY DEFAULT gen_random_uuid(),subscription_id uuid NOT NULL,subscription_charge_id uuid NOT NULL UNIQUE,team_id uuid NOT NULL,plan_code text NOT NULL,billing_market char(2) NOT NULL,currency char(3) NOT NULL,period_start timestamptz NOT NULL,period_end timestamptz NOT NULL,granted_units bigint NOT NULL,consumed_units bigint NOT NULL DEFAULT 0,created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now(),UNIQUE(subscription_id,period_start),FOREIGN KEY(subscription_id,team_id) REFERENCES team_subscriptions(id,team_id),FOREIGN KEY(subscription_charge_id,team_id) REFERENCES subscription_charges(id,team_id));
	CREATE TABLE product_rates(id uuid PRIMARY KEY,product text NOT NULL,meter text NOT NULL,billing_market char(2) NOT NULL,tier text NOT NULL,currency char(3) NOT NULL,cost_units bigint NOT NULL,effective_from timestamptz NOT NULL,effective_until timestamptz,created_at timestamptz NOT NULL DEFAULT now());
	CREATE TABLE sms_rates(id uuid PRIMARY KEY,destination_country char(2) NOT NULL,route_type text NOT NULL,tier text NOT NULL,currency char(3) NOT NULL,cost_units bigint NOT NULL,effective_from timestamptz NOT NULL,effective_until timestamptz,created_at timestamptz NOT NULL DEFAULT now());
	CREATE TABLE fx_rates(id uuid PRIMARY KEY,base_currency char(3) NOT NULL,quote_currency char(3) NOT NULL,rate numeric NOT NULL,effective_from timestamptz NOT NULL,effective_until timestamptz,created_at timestamptz NOT NULL DEFAULT now());
	CREATE TABLE allowance_policies(id uuid PRIMARY KEY DEFAULT gen_random_uuid(),product text NOT NULL,meter text NOT NULL,billing_market char(2) NOT NULL,tier text NOT NULL,included_quantity bigint NOT NULL,cadence text NOT NULL DEFAULT 'monthly',effective_from timestamptz NOT NULL,effective_until timestamptz,created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now());
	CREATE TABLE usage_allowances(id uuid PRIMARY KEY DEFAULT gen_random_uuid(),team_id uuid NOT NULL,allowance_policy_id uuid NOT NULL,product text NOT NULL,meter text NOT NULL,billing_market char(2) NOT NULL,tier text NOT NULL,period_start timestamptz NOT NULL,period_end timestamptz NOT NULL,included_quantity bigint NOT NULL,consumed_quantity bigint NOT NULL DEFAULT 0,created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now(),UNIQUE(team_id,product,meter,period_start,period_end));
	CREATE TABLE usage_authorizations(id uuid PRIMARY KEY DEFAULT gen_random_uuid(),team_id uuid NOT NULL,product text NOT NULL,meter text NOT NULL,reference_id text NOT NULL,usage_allowance_id uuid,sms_rate_id uuid,fx_rate_id uuid,product_rate_id uuid,billing_market char(2) NOT NULL,destination_country char(2),route_type text,total_quantity bigint NOT NULL,allowance_quantity bigint NOT NULL DEFAULT 0,billable_quantity bigint NOT NULL DEFAULT 0,unit_cost_units bigint NOT NULL DEFAULT 0,amount_units bigint NOT NULL DEFAULT 0,subscription_credit_id uuid,full_cost_units bigint NOT NULL DEFAULT 0,credit_consumed_units bigint NOT NULL DEFAULT 0,wallet_debit_units bigint NOT NULL DEFAULT 0,currency char(3) NOT NULL,tier text NOT NULL,priced_at timestamptz NOT NULL DEFAULT now(),created_at timestamptz NOT NULL DEFAULT now(),UNIQUE(team_id,product,meter,reference_id));
CREATE TABLE wallet_ledger(id uuid PRIMARY KEY DEFAULT gen_random_uuid(),team_id uuid NOT NULL REFERENCES teams(id),usage_authorization_id uuid,subscription_charge_id uuid,amount_units bigint NOT NULL,transaction_type text NOT NULL,reference_id text NOT NULL,created_at timestamptz NOT NULL DEFAULT now(),UNIQUE(team_id,transaction_type,reference_id),FOREIGN KEY(subscription_charge_id,team_id) REFERENCES subscription_charges(id,team_id));
	CREATE UNIQUE INDEX uq_wallet_ledger_subscription_charge ON wallet_ledger(subscription_charge_id) WHERE subscription_charge_id IS NOT NULL;
	CREATE TABLE outbox_events(id uuid PRIMARY KEY,subject text NOT NULL,aggregate_type text NOT NULL,aggregate_id uuid NOT NULL,payload jsonb NOT NULL,headers jsonb NOT NULL DEFAULT '{}',available_at timestamptz NOT NULL DEFAULT now(),attempts integer NOT NULL DEFAULT 0,locked_at timestamptz,locked_by text,published_at timestamptz,last_error text,created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now());
	`
