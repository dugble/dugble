CREATE TABLE IF NOT EXISTS payment_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL
        REFERENCES team_wallets(team_id)
        ON DELETE RESTRICT,
    provider TEXT NOT NULL,
    client_reference TEXT NOT NULL,
    currency CHAR(3) NOT NULL
        REFERENCES currencies(code)
        ON DELETE RESTRICT,
    amount_units BIGINT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    provider_transaction_id TEXT,
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_payment_transactions_id_team
        UNIQUE (id, team_id),
    CONSTRAINT uq_payment_transactions_provider_reference
        UNIQUE (provider, client_reference),
    CONSTRAINT chk_payment_transactions_provider
        CHECK (
            length(trim(provider)) > 0
            AND provider = lower(trim(provider))
            AND provider !~ '[[:space:]]'
        ),
    CONSTRAINT chk_payment_transactions_client_reference
        CHECK (length(trim(client_reference)) > 0),
    CONSTRAINT chk_payment_transactions_amount
        CHECK (amount_units > 0),
    CONSTRAINT chk_payment_transactions_status
        CHECK (status IN ('pending', 'paid', 'failed')),
    CONSTRAINT chk_payment_transactions_provider_transaction
        CHECK (
            provider_transaction_id IS NULL
            OR length(trim(provider_transaction_id)) > 0
        ),
    CONSTRAINT chk_payment_transactions_paid
        CHECK (
            (status = 'paid' AND paid_at IS NOT NULL)
            OR
            (status <> 'paid' AND paid_at IS NULL)
        )
);

CREATE INDEX IF NOT EXISTS idx_payment_transactions_team_created
    ON payment_transactions (team_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_payment_transactions_pending
    ON payment_transactions (created_at)
    WHERE status = 'pending';

CREATE UNIQUE INDEX IF NOT EXISTS uq_payment_transactions_provider_transaction
    ON payment_transactions (provider, provider_transaction_id)
    WHERE provider_transaction_id IS NOT NULL;
