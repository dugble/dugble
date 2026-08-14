CREATE TABLE IF NOT EXISTS outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subject TEXT NOT NULL,
    aggregate_type TEXT NOT NULL,
    aggregate_id UUID NOT NULL,
    payload JSONB NOT NULL,
    headers JSONB NOT NULL DEFAULT '{}'::jsonb,
    available_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ,
    attempts INTEGER NOT NULL DEFAULT 0,
    publish_failures INTEGER NOT NULL DEFAULT 0,
    redrive_count INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    quarantined_at TIMESTAMPTZ,
    quarantine_code TEXT,
    quarantine_reason TEXT,
    locked_at TIMESTAMPTZ,
    locked_by TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_outbox_subject CHECK (length(trim(subject)) > 0 AND subject !~ '[[:space:]]'),
    CONSTRAINT chk_outbox_aggregate_type CHECK (length(trim(aggregate_type)) > 0),
    CONSTRAINT chk_outbox_headers_object CHECK (jsonb_typeof(headers) = 'object'),
    CONSTRAINT chk_outbox_attempts_non_negative CHECK (attempts >= 0),
    CONSTRAINT chk_outbox_publish_failures_non_negative CHECK (publish_failures >= 0),
    CONSTRAINT chk_outbox_redrive_count_non_negative CHECK (redrive_count >= 0),
    CONSTRAINT chk_outbox_quarantine_state CHECK (
        (quarantined_at IS NULL AND quarantine_code IS NULL AND quarantine_reason IS NULL)
        OR (
            quarantined_at IS NOT NULL
            AND quarantine_code IS NOT NULL
            AND length(trim(quarantine_code)) > 0
            AND quarantine_reason IS NOT NULL
            AND length(trim(quarantine_reason)) > 0
        )
    ),
    CONSTRAINT chk_outbox_terminal_state CHECK (
        NOT (published_at IS NOT NULL AND quarantined_at IS NOT NULL)
    ),
    CONSTRAINT chk_outbox_lock_pair CHECK (
        (locked_at IS NULL AND locked_by IS NULL)
        OR (locked_at IS NOT NULL AND locked_by IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_outbox_events_pending
    ON outbox_events (available_at, created_at)
    WHERE published_at IS NULL AND quarantined_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_outbox_events_locked
    ON outbox_events (locked_at)
    WHERE published_at IS NULL AND quarantined_at IS NULL AND locked_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_outbox_events_aggregate
    ON outbox_events (aggregate_type, aggregate_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_outbox_events_quarantined
    ON outbox_events (quarantined_at DESC)
    WHERE quarantined_at IS NOT NULL;

CREATE TABLE IF NOT EXISTS outbox_event_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES outbox_events(id) ON DELETE CASCADE,
    action TEXT NOT NULL,
    reason_code TEXT,
    reason TEXT,
    publish_failures INTEGER NOT NULL DEFAULT 0,
    redrive_count INTEGER NOT NULL DEFAULT 0,
    actor_type TEXT NOT NULL,
    actor_id TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_outbox_event_actions_action CHECK (action IN ('quarantined', 'redriven')),
    CONSTRAINT chk_outbox_event_actions_reason_code CHECK (
        reason_code IS NULL OR length(trim(reason_code)) > 0
    ),
    CONSTRAINT chk_outbox_event_actions_reason CHECK (
        reason IS NULL OR length(trim(reason)) > 0
    ),
    CONSTRAINT chk_outbox_event_actions_publish_failures_non_negative CHECK (publish_failures >= 0),
    CONSTRAINT chk_outbox_event_actions_redrive_count_non_negative CHECK (redrive_count >= 0),
    CONSTRAINT chk_outbox_event_actions_actor_type CHECK (length(trim(actor_type)) > 0),
    CONSTRAINT chk_outbox_event_actions_actor_id CHECK (
        actor_id IS NULL OR length(trim(actor_id)) > 0
    ),
    CONSTRAINT chk_outbox_event_actions_metadata_object CHECK (jsonb_typeof(metadata) = 'object')
);

CREATE INDEX IF NOT EXISTS idx_outbox_event_actions_event
    ON outbox_event_actions (event_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_outbox_event_actions_action
    ON outbox_event_actions (action, created_at DESC);

CREATE TABLE IF NOT EXISTS processed_events (
    consumer_name TEXT NOT NULL,
    event_id UUID NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,

    PRIMARY KEY (consumer_name, event_id),
    CONSTRAINT chk_processed_events_consumer CHECK (length(trim(consumer_name)) > 0),
    CONSTRAINT chk_processed_events_metadata_object CHECK (jsonb_typeof(metadata) = 'object')
);
