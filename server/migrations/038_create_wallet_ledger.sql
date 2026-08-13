CREATE TABLE IF NOT EXISTS wallet_ledger (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL
        REFERENCES teams(id)
        ON DELETE RESTRICT,
    usage_authorization_id UUID,
    subscription_charge_id UUID,
    payment_transaction_id UUID,
    amount_units BIGINT NOT NULL,
    transaction_type TEXT NOT NULL,
    reference_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_wallet_ledger_reference
        UNIQUE (team_id, transaction_type, reference_id),
    CONSTRAINT fk_wallet_ledger_usage_authorization_same_team
        FOREIGN KEY (usage_authorization_id, team_id)
        REFERENCES usage_authorizations(id, team_id)
        ON DELETE RESTRICT,
    CONSTRAINT fk_wallet_ledger_subscription_charge_same_team
        FOREIGN KEY (subscription_charge_id, team_id)
        REFERENCES subscription_charges(id, team_id)
        ON DELETE RESTRICT,
    CONSTRAINT fk_wallet_ledger_payment_transaction_same_team
        FOREIGN KEY (payment_transaction_id, team_id)
        REFERENCES payment_transactions(id, team_id)
        ON DELETE RESTRICT,
    CONSTRAINT chk_wallet_ledger_amount
        CHECK (amount_units <> 0),
    CONSTRAINT chk_wallet_ledger_transaction_type
        CHECK (
            transaction_type IN (
                'deposit',
                'usage',
                'subscription',
                'refund',
                'expiry_wipe',
                'adjustment'
            )
        ),
    CONSTRAINT chk_wallet_ledger_charge_source
        CHECK (
            (
                transaction_type = 'deposit'
                AND usage_authorization_id IS NULL
                AND subscription_charge_id IS NULL
                AND payment_transaction_id IS NOT NULL
                AND amount_units > 0
            )
            OR
            (
                transaction_type = 'usage'
                AND usage_authorization_id IS NOT NULL
                AND subscription_charge_id IS NULL
                AND payment_transaction_id IS NULL
                AND amount_units < 0
            )
            OR
            (
                transaction_type = 'subscription'
                AND usage_authorization_id IS NULL
                AND subscription_charge_id IS NOT NULL
                AND payment_transaction_id IS NULL
                AND amount_units < 0
            )
            OR
            (
                transaction_type IN ('refund', 'expiry_wipe', 'adjustment')
                AND usage_authorization_id IS NULL
                AND subscription_charge_id IS NULL
                AND payment_transaction_id IS NULL
            )
        ),
    CONSTRAINT chk_wallet_ledger_reference
        CHECK (length(trim(reference_id)) > 0)
);

CREATE INDEX IF NOT EXISTS idx_wallet_ledger_team_created
    ON wallet_ledger (team_id, created_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS uq_wallet_ledger_usage_authorization
    ON wallet_ledger (usage_authorization_id)
    WHERE usage_authorization_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_wallet_ledger_subscription_charge
    ON wallet_ledger (subscription_charge_id)
    WHERE subscription_charge_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_wallet_ledger_payment_transaction
    ON wallet_ledger (payment_transaction_id)
    WHERE payment_transaction_id IS NOT NULL;
