INSERT INTO currencies (
    code,
    minor_unit,
    is_enabled
)
VALUES
    ('USD', 2, true),
    ('GHS', 2, true),
    ('KES', 2, true)
ON CONFLICT (code) DO UPDATE
SET
    minor_unit = EXCLUDED.minor_unit,
    is_enabled = EXCLUDED.is_enabled;

INSERT INTO billing_markets (
    code,
    currency,
    is_enabled
)
VALUES
    ('GH', 'GHS', true),
    ('KE', 'KES', true)
ON CONFLICT (code) DO UPDATE
SET
    currency = EXCLUDED.currency,
    is_enabled = EXCLUDED.is_enabled;

INSERT INTO product_rates (
    product,
    meter,
    billing_market,
    tier,
    currency,
    cost_units,
    effective_from
)
VALUES
    ('email', 'email_recipient', 'GH', 'growth',     'GHS',   9392, '2026-08-01T00:00:00Z'),
    ('email', 'email_recipient', 'GH', 'scale',      'GHS',   7044, '2026-08-01T00:00:00Z'),
    ('email', 'email_recipient', 'GH', 'enterprise', 'GHS',   4696, '2026-08-01T00:00:00Z'),
    ('email', 'email_recipient', 'KE', 'growth',     'KES', 103504, '2026-08-01T00:00:00Z'),
    ('email', 'email_recipient', 'KE', 'scale',      'KES',  77628, '2026-08-01T00:00:00Z'),
    ('email', 'email_recipient', 'KE', 'enterprise', 'KES',  51752, '2026-08-01T00:00:00Z');

INSERT INTO fx_rates (
    base_currency,
    quote_currency,
    rate,
    effective_from
)
VALUES
    ('USD', 'GHS',  11.74000000, '2026-08-01T00:00:00Z'),
    ('USD', 'KES', 129.38000000, '2026-08-01T00:00:00Z');

INSERT INTO sms_rates (
    destination_country,
    route_type,
    tier,
    currency,
    cost_units,
    effective_from
)
VALUES
    -- Ghana international destination (+233), priced in USD.
    ('GH', 'intl', 'growth',     'USD', 15000, '2026-08-01T00:00:00Z'),
    ('GH', 'intl', 'scale',      'USD', 12000, '2026-08-01T00:00:00Z'),
    ('GH', 'intl', 'enterprise', 'USD',  9500, '2026-08-01T00:00:00Z'),

    -- Kenya international destination (+254), priced in USD.
    ('KE', 'intl', 'growth',     'USD', 18000, '2026-08-01T00:00:00Z'),
    ('KE', 'intl', 'scale',      'USD', 15000, '2026-08-01T00:00:00Z'),
    ('KE', 'intl', 'enterprise', 'USD', 12500, '2026-08-01T00:00:00Z'),

    -- Ghana local destination (+233), priced in GHS.
    ('GH', 'local', 'growth',     'GHS', 65000, '2026-08-01T00:00:00Z'),
    ('GH', 'local', 'scale',      'GHS', 55000, '2026-08-01T00:00:00Z'),
    ('GH', 'local', 'enterprise', 'GHS', 45000, '2026-08-01T00:00:00Z'),

    -- Kenya local destination (+254), priced in KES.
    ('KE', 'local', 'growth',     'KES', 950000, '2026-08-01T00:00:00Z'),
    ('KE', 'local', 'scale',      'KES', 800000, '2026-08-01T00:00:00Z'),
    ('KE', 'local', 'enterprise', 'KES', 650000, '2026-08-01T00:00:00Z');

INSERT INTO allowance_policies (
    product,
    meter,
    billing_market,
    tier,
    included_quantity,
    cadence,
    effective_from
)
VALUES
    (
        'email',
        'email_recipient',
        'GH',
        'growth',
        1000,
        'monthly',
        '2026-08-01T00:00:00Z'
    ),
    (
        'email',
        'email_recipient',
        'KE',
        'growth',
        1000,
        'monthly',
        '2026-08-01T00:00:00Z'
    );
