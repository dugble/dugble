ALTER TABLE sender_ids
    ADD CONSTRAINT uq_sender_ids_id_team UNIQUE (id, team_id);

CREATE TABLE sms_campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft',
    segment_id UUID NOT NULL,
    sender_id UUID NOT NULL,
    body TEXT NOT NULL,
    scheduled_at TIMESTAMPTZ,
    queued_at TIMESTAMPTZ,
    canceled_at TIMESTAMPTZ,
    materialized_at TIMESTAMPTZ,
    sent_at TIMESTAMPTZ,
    audience_count BIGINT NOT NULL DEFAULT 0,
    eligible_count BIGINT NOT NULL DEFAULT 0,
    excluded_count BIGINT NOT NULL DEFAULT 0,
    failed_count BIGINT NOT NULL DEFAULT 0,
    estimated_segments BIGINT NOT NULL DEFAULT 0,
    estimated_cost_units BIGINT NOT NULL DEFAULT 0,
    estimated_billable_cost_units BIGINT NOT NULL DEFAULT 0,
    preflight_allowance_segments BIGINT NOT NULL DEFAULT 0,
    actual_segments BIGINT NOT NULL DEFAULT 0,
    actual_charge_units BIGINT NOT NULL DEFAULT 0,
    currency CHAR(3),
    preflight_balance_units BIGINT,
    preflight_at TIMESTAMPTZ,
    rate_limit_per_second INTEGER NOT NULL DEFAULT 10,
    daily_send_limit INTEGER,
    revision BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_sms_campaigns_id_team UNIQUE (id, team_id),
    CONSTRAINT fk_sms_campaigns_segment_team FOREIGN KEY (segment_id, team_id)
        REFERENCES segments(id, team_id) ON DELETE RESTRICT,
    CONSTRAINT fk_sms_campaigns_sender_team FOREIGN KEY (sender_id, team_id)
        REFERENCES sender_ids(id, team_id) ON DELETE RESTRICT,
    CONSTRAINT chk_sms_campaigns_name CHECK (length(btrim(name)) > 0),
    CONSTRAINT chk_sms_campaigns_body CHECK (length(btrim(body)) > 0 AND length(body) <= 1600),
    CONSTRAINT chk_sms_campaigns_status CHECK (
        status IN ('draft', 'scheduled', 'queued', 'materializing', 'estimating', 'sending', 'sent', 'failed', 'canceled')
    ),
    CONSTRAINT chk_sms_campaigns_counts CHECK (
        audience_count >= 0 AND eligible_count >= 0 AND excluded_count >= 0
        AND audience_count = eligible_count + excluded_count
    ),
    CONSTRAINT chk_sms_campaigns_failed_count CHECK (failed_count >= 0),
    CONSTRAINT chk_sms_campaigns_costs CHECK (
        estimated_segments >= 0 AND estimated_cost_units >= 0
        AND estimated_billable_cost_units >= 0 AND preflight_allowance_segments >= 0
        AND actual_segments >= 0 AND actual_charge_units >= 0
        AND (preflight_balance_units IS NULL OR preflight_balance_units >= 0)
    ),
    CONSTRAINT chk_sms_campaigns_rate_limit CHECK (rate_limit_per_second BETWEEN 1 AND 1000),
    CONSTRAINT chk_sms_campaigns_daily_limit CHECK (daily_send_limit IS NULL OR daily_send_limit > 0),
    CONSTRAINT chk_sms_campaigns_revision CHECK (revision > 0),
    CONSTRAINT chk_sms_campaigns_schedule CHECK (
        (status = 'draft' AND scheduled_at IS NULL AND queued_at IS NULL AND canceled_at IS NULL)
        OR (status = 'scheduled' AND scheduled_at IS NOT NULL AND queued_at IS NULL AND canceled_at IS NULL)
        OR (status IN ('queued', 'materializing', 'estimating', 'sending') AND queued_at IS NOT NULL AND canceled_at IS NULL)
        OR (status = 'sent' AND queued_at IS NOT NULL AND sent_at IS NOT NULL AND canceled_at IS NULL)
        OR (status = 'failed' AND queued_at IS NOT NULL AND canceled_at IS NULL)
        OR (status = 'canceled' AND canceled_at IS NOT NULL)
    )
);

CREATE INDEX idx_sms_campaigns_team_created ON sms_campaigns (team_id, created_at DESC, id DESC);
CREATE INDEX idx_sms_campaigns_due ON sms_campaigns (scheduled_at, id) WHERE status = 'scheduled';
CREATE INDEX idx_sms_campaigns_rate_control ON sms_campaigns (queued_at, id) WHERE status = 'sending';

CREATE TABLE sms_campaign_recipients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL,
    campaign_id UUID NOT NULL,
    contact_id UUID,
    phone TEXT,
    phone_country CHAR(2),
    contact_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    status TEXT NOT NULL,
    exclusion_reason TEXT,
    sms_message_id UUID,
    rendered_body TEXT,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    failure_code TEXT,
    failure_message TEXT,
    encoding TEXT,
    estimated_segments INTEGER,
    estimated_unit_cost_units BIGINT,
    estimated_cost_units BIGINT,
    actual_segments INTEGER,
    actual_charge_units BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    queued_at TIMESTAMPTZ,

    CONSTRAINT fk_sms_campaign_recipients_campaign FOREIGN KEY (campaign_id, team_id)
        REFERENCES sms_campaigns(id, team_id) ON DELETE CASCADE,
    CONSTRAINT fk_sms_campaign_recipients_contact FOREIGN KEY (contact_id)
        REFERENCES contacts(id) ON DELETE SET NULL,
    CONSTRAINT fk_sms_campaign_recipients_message FOREIGN KEY (sms_message_id)
        REFERENCES sms_messages(id) ON DELETE SET NULL,
    CONSTRAINT uq_sms_campaign_recipients_contact UNIQUE (campaign_id, contact_id),
    CONSTRAINT chk_sms_campaign_recipients_phone CHECK (
        (phone IS NULL AND phone_country IS NULL)
        OR (phone IS NOT NULL AND phone_country IS NOT NULL
            AND phone ~ '^\+[1-9][0-9]{7,14}$' AND phone_country ~ '^[A-Z]{2}$')
    ),
    CONSTRAINT chk_sms_campaign_recipients_status CHECK (
        status IN ('pending', 'processing', 'excluded', 'queued', 'failed')
    ),
    CONSTRAINT chk_sms_campaign_recipients_attempts CHECK (attempt_count >= 0),
    CONSTRAINT chk_sms_campaign_recipients_exclusion CHECK (
        (status = 'excluded' AND exclusion_reason IS NOT NULL)
        OR (status <> 'excluded' AND exclusion_reason IS NULL)
    ),
    CONSTRAINT chk_sms_campaign_recipients_rendered_body CHECK (
        rendered_body IS NULL OR (length(btrim(rendered_body)) > 0 AND length(rendered_body) <= 1600)
    ),
    CONSTRAINT chk_sms_campaign_recipients_encoding CHECK (
        encoding IS NULL OR encoding IN ('GSM-7', 'UCS-2')
    ),
    CONSTRAINT chk_sms_campaign_recipients_costs CHECK (
        (estimated_segments IS NULL OR estimated_segments > 0)
        AND (estimated_unit_cost_units IS NULL OR estimated_unit_cost_units >= 0)
        AND (estimated_cost_units IS NULL OR estimated_cost_units >= 0)
        AND (actual_segments IS NULL OR actual_segments > 0)
        AND (actual_charge_units IS NULL OR actual_charge_units >= 0)
    )
);

CREATE INDEX idx_sms_campaign_recipients_campaign_status
    ON sms_campaign_recipients (campaign_id, status, created_at, id);
CREATE INDEX idx_sms_campaign_recipients_fanout
    ON sms_campaign_recipients (next_attempt_at, created_at, id)
    WHERE status = 'pending';
CREATE INDEX idx_sms_campaign_recipients_estimation
    ON sms_campaign_recipients (created_at, id)
    WHERE status = 'pending' AND estimated_segments IS NULL;

CREATE TABLE sms_consent_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    contact_id UUID REFERENCES contacts(id) ON DELETE SET NULL,
    phone TEXT NOT NULL,
    action TEXT NOT NULL,
    source TEXT NOT NULL,
    source_id TEXT,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_sms_consent_events_phone CHECK (phone ~ '^\+[1-9][0-9]{7,14}$'),
    CONSTRAINT chk_sms_consent_events_action CHECK (action IN ('opted_out')),
    CONSTRAINT chk_sms_consent_events_source CHECK (source IN ('api', 'manual', 'import')),
    CONSTRAINT chk_sms_consent_events_source_id CHECK (
        source_id IS NULL OR length(btrim(source_id)) > 0
    )
);

CREATE INDEX idx_sms_consent_events_team_phone_recorded
    ON sms_consent_events (team_id, phone, recorded_at DESC);
CREATE UNIQUE INDEX uq_sms_consent_events_source
    ON sms_consent_events (team_id, source, source_id)
    WHERE source_id IS NOT NULL;
