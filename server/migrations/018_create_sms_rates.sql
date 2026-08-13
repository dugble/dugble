CREATE TABLE IF NOT EXISTS sms_rates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    destination_country CHAR(2) NOT NULL,
    route_type TEXT NOT NULL,
    tier TEXT NOT NULL,
    currency CHAR(3) NOT NULL
        REFERENCES currencies(code)
        ON DELETE RESTRICT,
    cost_units BIGINT NOT NULL,

    effective_from TIMESTAMPTZ NOT NULL,
    effective_until TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_sms_rates_audit
        UNIQUE (
            id,
            destination_country,
            route_type,
            tier,
            currency,
            cost_units
        ),

    CONSTRAINT chk_sms_rates_destination_country
        CHECK (destination_country ~ '^[A-Z]{2}$'),

    CONSTRAINT chk_sms_rates_route_type
        CHECK (route_type IN ('local', 'intl')),

    CONSTRAINT chk_sms_rates_international_currency
        CHECK (route_type <> 'intl' OR currency = 'USD'),

    CONSTRAINT chk_sms_rates_tier
        CHECK (tier IN ('growth', 'scale', 'enterprise')),

    CONSTRAINT chk_sms_rates_cost
        CHECK (cost_units > 0),

    CONSTRAINT chk_sms_rates_period
        CHECK (
            effective_until IS NULL
            OR effective_until > effective_from
        )
);

ALTER TABLE sms_rates
ADD CONSTRAINT ex_sms_rates_no_overlap
EXCLUDE USING gist (
    destination_country WITH =,
    route_type WITH =,
    tier WITH =,
    tstzrange(
        effective_from,
        COALESCE(effective_until, 'infinity'::timestamptz),
        '[)'
    ) WITH &&
);

CREATE INDEX IF NOT EXISTS idx_sms_rates_lookup
    ON sms_rates (
        destination_country,
        route_type,
        tier,
        effective_from DESC
    );

CREATE TABLE IF NOT EXISTS fx_rates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    base_currency CHAR(3) NOT NULL
        REFERENCES currencies(code)
        ON DELETE RESTRICT,
    quote_currency CHAR(3) NOT NULL
        REFERENCES currencies(code)
        ON DELETE RESTRICT,
    rate NUMERIC(20, 8) NOT NULL,

    effective_from TIMESTAMPTZ NOT NULL,
    effective_until TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_fx_rates_audit
        UNIQUE (
            id,
            base_currency,
            quote_currency,
            rate
        ),

    CONSTRAINT chk_fx_rates_currency_pair
        CHECK (base_currency <> quote_currency),

    CONSTRAINT chk_fx_rates_rate
        CHECK (rate > 0),

    CONSTRAINT chk_fx_rates_period
        CHECK (
            effective_until IS NULL
            OR effective_until > effective_from
        )
);

ALTER TABLE fx_rates
ADD CONSTRAINT ex_fx_rates_no_overlap
EXCLUDE USING gist (
    base_currency WITH =,
    quote_currency WITH =,
    tstzrange(
        effective_from,
        COALESCE(effective_until, 'infinity'::timestamptz),
        '[)'
    ) WITH &&
);

CREATE INDEX IF NOT EXISTS idx_fx_rates_lookup
    ON fx_rates (
        base_currency,
        quote_currency,
        effective_from DESC
    );
