ALTER TABLE outbox_events
    ADD COLUMN publish_failures INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN quarantined_at TIMESTAMPTZ,
    ADD COLUMN quarantine_code TEXT,
    ADD COLUMN quarantine_reason TEXT,
    ADD CONSTRAINT chk_outbox_publish_failures_non_negative CHECK (publish_failures >= 0),
    ADD CONSTRAINT chk_outbox_quarantine_state CHECK (
        (quarantined_at IS NULL AND quarantine_code IS NULL AND quarantine_reason IS NULL)
        OR (
            quarantined_at IS NOT NULL
            AND quarantine_code IS NOT NULL
            AND length(trim(quarantine_code)) > 0
            AND quarantine_reason IS NOT NULL
            AND length(trim(quarantine_reason)) > 0
        )
    ),
    ADD CONSTRAINT chk_outbox_terminal_state CHECK (
        NOT (published_at IS NOT NULL AND quarantined_at IS NOT NULL)
    );

DROP INDEX IF EXISTS idx_outbox_events_pending;
CREATE INDEX idx_outbox_events_pending
    ON outbox_events (available_at, created_at)
    WHERE published_at IS NULL AND quarantined_at IS NULL;

DROP INDEX IF EXISTS idx_outbox_events_locked;
CREATE INDEX idx_outbox_events_locked
    ON outbox_events (locked_at)
    WHERE published_at IS NULL AND quarantined_at IS NULL AND locked_at IS NOT NULL;

CREATE INDEX idx_outbox_events_quarantined
    ON outbox_events (quarantined_at DESC)
    WHERE quarantined_at IS NOT NULL;
