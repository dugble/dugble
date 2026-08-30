ALTER TABLE broadcasts
    ADD COLUMN from_email TEXT,
    ADD COLUMN from_name TEXT,
    ADD COLUMN reply_to_email TEXT,
    ADD COLUMN subject TEXT,
    ADD COLUMN preview_text TEXT,
    ADD COLUMN html_body TEXT,
    ADD COLUMN text_body TEXT;

-- Backfill the message snapshot that a legacy broadcast would send. Broadcasts
-- that already started execution keep their pinned version. Draft/scheduled
-- broadcasts prefer the published version, matching the legacy send path, and
-- fall back to the current version so every valid legacy broadcast can migrate.
UPDATE broadcasts AS broadcast
SET from_email = version.from_email,
    from_name = version.from_name,
    reply_to_email = version.reply_to_email,
    subject = version.subject,
    html_body = version.html_body,
    text_body = version.text_body
FROM message_templates AS template
JOIN message_template_versions AS version
  ON version.id = COALESCE(template.published_version_id, template.current_version_id)
 AND version.template_id = template.id
 AND version.team_id = template.team_id
WHERE broadcast.template_version_id IS NULL
  AND broadcast.template_id = template.id
  AND broadcast.team_id = template.team_id;

UPDATE broadcasts AS broadcast
SET from_email = version.from_email,
    from_name = version.from_name,
    reply_to_email = version.reply_to_email,
    subject = version.subject,
    html_body = version.html_body,
    text_body = version.text_body
FROM message_template_versions AS version
WHERE broadcast.template_version_id IS NOT NULL
  AND version.id = broadcast.template_version_id
  AND version.template_id = broadcast.template_id
  AND version.team_id = broadcast.team_id;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM broadcasts
        WHERE subject IS NULL OR html_body IS NULL
    ) THEN
        RAISE EXCEPTION 'cannot migrate broadcasts without resolvable message content';
    END IF;
END
$$;

ALTER TABLE broadcasts
    ALTER COLUMN subject SET NOT NULL,
    ALTER COLUMN html_body SET NOT NULL,
    ALTER COLUMN template_id DROP NOT NULL;

ALTER TABLE broadcasts
    ADD CONSTRAINT chk_broadcasts_subject_not_empty
        CHECK (length(btrim(subject)) > 0),
    ADD CONSTRAINT chk_broadcasts_html_not_empty
        CHECK (length(btrim(html_body)) > 0);
