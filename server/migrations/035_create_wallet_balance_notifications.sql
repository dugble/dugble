CREATE TABLE IF NOT EXISTS wallet_balance_notifications (
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    level TEXT NOT NULL,
    balance_units BIGINT NOT NULL,
    notified_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (team_id, level),
    CONSTRAINT chk_wallet_balance_notification_level
        CHECK (level IN ('low', 'exhausted')),
    CONSTRAINT chk_wallet_balance_notification_balance
        CHECK (balance_units >= 0)
);
