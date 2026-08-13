CREATE TABLE channel_suppressions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    channel TEXT NOT NULL,
    address TEXT NOT NULL,
    normalized_address TEXT NOT NULL,
    reason TEXT NOT NULL,
    origin TEXT NOT NULL DEFAULT 'manual',
    source_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_channel_suppressions_id_team UNIQUE (id, team_id),
    CONSTRAINT uq_channel_suppressions_team_address UNIQUE (team_id, channel, normalized_address),
    CONSTRAINT chk_channel_suppressions_channel CHECK (channel IN ('email', 'sms')),
    CONSTRAINT chk_channel_suppressions_address CHECK (
        length(btrim(address)) > 0 AND length(btrim(normalized_address)) > 0
    ),
    CONSTRAINT chk_channel_suppressions_normalized_address CHECK (
        (channel = 'email' AND normalized_address = lower(btrim(normalized_address)))
        OR
        (channel = 'sms' AND normalized_address ~ '^\+[1-9][0-9]{7,14}$')
    ),
    CONSTRAINT chk_channel_suppressions_reason CHECK (
        reason IN ('manual', 'user_opt_out', 'bounce', 'complaint', 'invalid_address')
    ),
    CONSTRAINT chk_channel_suppressions_origin CHECK (
        origin IN ('manual', 'api', 'import', 'provider')
    ),
    CONSTRAINT chk_channel_suppressions_source_id CHECK (
        source_id IS NULL OR length(btrim(source_id)) > 0
    )
);

CREATE INDEX idx_channel_suppressions_team_channel_created
    ON channel_suppressions (team_id, channel, created_at DESC, id DESC);
