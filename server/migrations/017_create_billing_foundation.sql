CREATE TABLE IF NOT EXISTS currencies (
    code CHAR(3) PRIMARY KEY,
    minor_unit SMALLINT NOT NULL,
    is_enabled BOOLEAN NOT NULL DEFAULT true,

    CONSTRAINT chk_currencies_code
        CHECK (code ~ '^[A-Z]{3}$'),

    CONSTRAINT chk_currencies_minor_unit
        CHECK (minor_unit BETWEEN 0 AND 6)
);

CREATE TABLE IF NOT EXISTS billing_markets (
    code CHAR(2) PRIMARY KEY,
    currency CHAR(3) NOT NULL
        REFERENCES currencies(code)
        ON DELETE RESTRICT,
    is_enabled BOOLEAN NOT NULL DEFAULT true,

    CONSTRAINT chk_billing_markets_code
        CHECK (code ~ '^[A-Z]{2}$'),

    CONSTRAINT uq_billing_markets_code_currency
        UNIQUE (code, currency)
);

CREATE TABLE IF NOT EXISTS team_wallets (
    team_id UUID PRIMARY KEY,
    billing_market CHAR(2) NOT NULL,
    currency CHAR(3) NOT NULL,
    balance_units BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_team_wallets_team_market_currency
        UNIQUE (team_id, billing_market, currency),

    CONSTRAINT fk_wallet_team_market
        FOREIGN KEY (team_id, billing_market)
        REFERENCES teams (id, market_code)
        ON DELETE CASCADE,

    CONSTRAINT fk_team_wallets_billing_market_currency
        FOREIGN KEY (billing_market, currency)
        REFERENCES billing_markets (code, currency)
        ON DELETE RESTRICT,

    CONSTRAINT chk_team_wallets_balance
        CHECK (balance_units >= 0)
);
