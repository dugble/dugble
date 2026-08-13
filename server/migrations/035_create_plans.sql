CREATE TABLE IF NOT EXISTS plans (
    code TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    is_enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_plans_code
        CHECK (code IN ('growth', 'scale', 'enterprise')),
    CONSTRAINT chk_plans_name
        CHECK (length(trim(name)) > 0)
);

CREATE TABLE IF NOT EXISTS plan_prices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_code TEXT NOT NULL
        REFERENCES plans(code)
        ON DELETE RESTRICT,
    billing_market CHAR(2) NOT NULL,
    currency CHAR(3) NOT NULL,
    amount_units BIGINT NOT NULL,
    billing_interval TEXT NOT NULL DEFAULT 'monthly',
    effective_from TIMESTAMPTZ NOT NULL,
    effective_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_plan_prices_audit
        UNIQUE (
            id,
            plan_code,
            billing_market,
            currency,
            amount_units
        ),
    CONSTRAINT fk_plan_prices_market_currency
        FOREIGN KEY (billing_market, currency)
        REFERENCES billing_markets(code, currency)
        ON DELETE RESTRICT,
    CONSTRAINT chk_plan_prices_amount
        CHECK (amount_units >= 0),
    CONSTRAINT chk_plan_prices_interval
        CHECK (billing_interval = 'monthly'),
    CONSTRAINT chk_plan_prices_period
        CHECK (
            effective_from = (
                date_trunc('month', effective_from AT TIME ZONE 'UTC')
                AT TIME ZONE 'UTC'
            )
            AND (
                effective_until IS NULL
                OR (
                    effective_until > effective_from
                    AND effective_until = (
                        date_trunc('month', effective_until AT TIME ZONE 'UTC')
                        AT TIME ZONE 'UTC'
                    )
                )
            )
        )
);

ALTER TABLE plan_prices
ADD CONSTRAINT ex_plan_prices_no_overlap
EXCLUDE USING gist (
    plan_code WITH =,
    billing_market WITH =,
    billing_interval WITH =,
    tstzrange(
        effective_from,
        COALESCE(effective_until, 'infinity'::timestamptz),
        '[)'
    ) WITH &&
);

CREATE INDEX IF NOT EXISTS idx_plan_prices_lookup
    ON plan_prices (
        plan_code,
        billing_market,
        billing_interval,
        effective_from DESC
    );

INSERT INTO plans (code, name)
VALUES
    ('growth', 'Growth'),
    ('scale', 'Scale'),
    ('enterprise', 'Enterprise')
ON CONFLICT (code) DO UPDATE
SET
    name = EXCLUDED.name,
    is_enabled = true,
    updated_at = now();

INSERT INTO plan_prices (
    plan_code,
    billing_market,
    currency,
    amount_units,
    billing_interval,
    effective_from
)
VALUES
    -- Ghana
    ('growth',     'GH', 'GHS',    99000000, 'monthly', '2026-08-01T00:00:00Z'),
    ('scale',      'GH', 'GHS',   499000000, 'monthly', '2026-08-01T00:00:00Z'),
    ('enterprise', 'GH', 'GHS',  1499000000, 'monthly', '2026-08-01T00:00:00Z'),

    -- Kenya
    ('growth',     'KE', 'KES',  1199000000, 'monthly', '2026-08-01T00:00:00Z'),
    ('scale',      'KE', 'KES',  5999000000, 'monthly', '2026-08-01T00:00:00Z'),
    ('enterprise', 'KE', 'KES', 17999000000, 'monthly', '2026-08-01T00:00:00Z');
