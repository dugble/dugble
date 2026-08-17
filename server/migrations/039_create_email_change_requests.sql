CREATE TABLE IF NOT EXISTS email_change_requests (
    user_id UUID PRIMARY KEY
        REFERENCES users(id)
        ON DELETE CASCADE,
    pending_email TEXT NOT NULL,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT chk_email_change_requests_pending_email
        CHECK (length(trim(pending_email)) > 0),
    CONSTRAINT chk_email_change_requests_expiry
        CHECK (expires_at > requested_at)
);

CREATE INDEX IF NOT EXISTS idx_email_change_requests_expires_at
    ON email_change_requests (expires_at);
