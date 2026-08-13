CREATE TABLE IF NOT EXISTS team_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL UNIQUE
        REFERENCES teams(id)
        ON DELETE RESTRICT,
    plan_code TEXT NOT NULL
        REFERENCES plans(code)
        ON DELETE RESTRICT,
    status TEXT NOT NULL DEFAULT 'active',
    current_period_start TIMESTAMPTZ NOT NULL,
    current_period_end TIMESTAMPTZ NOT NULL,
    pending_plan_code TEXT
        REFERENCES plans(code)
        ON DELETE RESTRICT,
    pending_plan_effective_at TIMESTAMPTZ,
    cancel_at_period_end BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_team_subscriptions_id_team
        UNIQUE (id, team_id),
    CONSTRAINT chk_team_subscriptions_status
        CHECK (status IN ('active', 'past_due', 'canceled')),
    CONSTRAINT chk_team_subscriptions_period
        CHECK (
            current_period_start = (
                date_trunc('month', current_period_start AT TIME ZONE 'UTC')
                AT TIME ZONE 'UTC'
            )
            AND current_period_end = (
                date_trunc('month', current_period_start AT TIME ZONE 'UTC')
                + interval '1 month'
            ) AT TIME ZONE 'UTC'
        ),
    CONSTRAINT chk_team_subscriptions_pending_plan
        CHECK (
            (
                pending_plan_code IS NULL
                AND pending_plan_effective_at IS NULL
            )
            OR
            (
                pending_plan_code IS NOT NULL
                AND pending_plan_effective_at = current_period_end
                AND pending_plan_code <> plan_code
            )
        )
);

CREATE INDEX IF NOT EXISTS idx_team_subscriptions_due
    ON team_subscriptions (current_period_end, team_id)
    WHERE status IN ('active', 'past_due');

CREATE TABLE IF NOT EXISTS subscription_charges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id UUID NOT NULL,
    team_id UUID NOT NULL
        REFERENCES teams(id)
        ON DELETE RESTRICT,
    plan_price_id UUID NOT NULL,
    plan_code TEXT NOT NULL,
    billing_market CHAR(2) NOT NULL,
    currency CHAR(3) NOT NULL,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    amount_units BIGINT NOT NULL,
    status TEXT NOT NULL,
    failure_code TEXT,
    attempt_count INTEGER NOT NULL DEFAULT 1,
    last_attempted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    applied_at TIMESTAMPTZ,
    reference_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_subscription_charges_id_team
        UNIQUE (id, team_id),
    CONSTRAINT uq_subscription_charges_period
        UNIQUE (subscription_id, period_start),
    CONSTRAINT uq_subscription_charges_reference
        UNIQUE (team_id, reference_id),
    CONSTRAINT fk_subscription_charges_subscription_team
        FOREIGN KEY (subscription_id, team_id)
        REFERENCES team_subscriptions(id, team_id)
        ON DELETE RESTRICT,
    CONSTRAINT fk_subscription_charges_price_audit
        FOREIGN KEY (
            plan_price_id,
            plan_code,
            billing_market,
            currency,
            amount_units
        )
        REFERENCES plan_prices (
            id,
            plan_code,
            billing_market,
            currency,
            amount_units
        )
        ON DELETE RESTRICT,
    CONSTRAINT chk_subscription_charges_period
        CHECK (
            period_start = (
                date_trunc('month', period_start AT TIME ZONE 'UTC')
                AT TIME ZONE 'UTC'
            )
            AND period_end = (
                date_trunc('month', period_start AT TIME ZONE 'UTC')
                + interval '1 month'
            ) AT TIME ZONE 'UTC'
        ),
    CONSTRAINT chk_subscription_charges_amount
        CHECK (amount_units >= 0),
    CONSTRAINT chk_subscription_charges_status
        CHECK (status IN ('applied', 'failed')),
    CONSTRAINT chk_subscription_charges_attempt_count
        CHECK (attempt_count > 0),
    CONSTRAINT chk_subscription_charges_outcome
        CHECK (
            (status = 'applied' AND failure_code IS NULL AND applied_at IS NOT NULL)
            OR
            (status = 'failed' AND length(trim(failure_code)) > 0 AND applied_at IS NULL)
        ),
    CONSTRAINT chk_subscription_charges_reference
        CHECK (length(trim(reference_id)) > 0)
);

CREATE INDEX IF NOT EXISTS idx_subscription_charges_team_created
    ON subscription_charges (team_id, created_at DESC);

CREATE TABLE IF NOT EXISTS subscription_credits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id UUID NOT NULL,
    subscription_charge_id UUID NOT NULL,
    team_id UUID NOT NULL,
    plan_code TEXT NOT NULL,
    billing_market CHAR(2) NOT NULL,
    currency CHAR(3) NOT NULL,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    granted_units BIGINT NOT NULL,
    consumed_units BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_subscription_credits_id_team
        UNIQUE (id, team_id),
    CONSTRAINT uq_subscription_credits_subscription_period
        UNIQUE (subscription_id, period_start),
    CONSTRAINT uq_subscription_credits_charge
        UNIQUE (subscription_charge_id),
    CONSTRAINT fk_subscription_credits_subscription_team
        FOREIGN KEY (subscription_id, team_id)
        REFERENCES team_subscriptions(id, team_id)
        ON DELETE RESTRICT,
    CONSTRAINT fk_subscription_credits_charge_team
        FOREIGN KEY (subscription_charge_id, team_id)
        REFERENCES subscription_charges(id, team_id)
        ON DELETE RESTRICT,
    CONSTRAINT fk_subscription_credits_plan
        FOREIGN KEY (plan_code)
        REFERENCES plans(code)
        ON DELETE RESTRICT,
    CONSTRAINT fk_subscription_credits_market_currency
        FOREIGN KEY (billing_market, currency)
        REFERENCES billing_markets(code, currency)
        ON DELETE RESTRICT,
    CONSTRAINT chk_subscription_credits_period
        CHECK (
            period_start = (
                date_trunc('month', period_start AT TIME ZONE 'UTC')
                AT TIME ZONE 'UTC'
            )
            AND period_end = (
                date_trunc('month', period_start AT TIME ZONE 'UTC')
                + interval '1 month'
            ) AT TIME ZONE 'UTC'
        ),
    CONSTRAINT chk_subscription_credits_units
        CHECK (
            granted_units > 0
            AND consumed_units >= 0
            AND consumed_units <= granted_units
        )
);

CREATE INDEX IF NOT EXISTS idx_subscription_credits_team_period
    ON subscription_credits (team_id, period_start DESC, period_end DESC);

ALTER TABLE usage_authorizations
ADD CONSTRAINT fk_usage_authorizations_subscription_credit_team
FOREIGN KEY (subscription_credit_id, team_id)
REFERENCES subscription_credits(id, team_id)
ON DELETE RESTRICT;
