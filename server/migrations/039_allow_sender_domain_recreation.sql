-- Preserve disabled sender-domain rows for history without reserving the name forever.
-- Only one active ownership record may exist for a normalized domain at a time.
ALTER TABLE domains
    DROP CONSTRAINT IF EXISTS uq_domains_normalized_name;

CREATE UNIQUE INDEX IF NOT EXISTS uq_domains_active_normalized_name
    ON domains (normalized_name)
    WHERE disabled_at IS NULL;
