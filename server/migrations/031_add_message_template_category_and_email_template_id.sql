CREATE TYPE message_template_category AS ENUM (
    'otp',
    'welcome',
    'receipt',
    'alert',
    'notification',
    'custom'
);

ALTER TABLE message_templates
    ADD COLUMN category message_template_category NOT NULL DEFAULT 'custom';

ALTER TABLE email_messages
    ADD COLUMN template_id UUID;

ALTER TABLE email_messages
    ADD CONSTRAINT fk_email_messages_template_team
        FOREIGN KEY (template_id, team_id)
        REFERENCES message_templates (id, team_id)
        ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_email_messages_template_created
    ON email_messages (team_id, template_id, created_at DESC)
    WHERE template_id IS NOT NULL;
