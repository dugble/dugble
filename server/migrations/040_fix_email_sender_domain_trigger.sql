CREATE OR REPLACE FUNCTION enforce_email_sender_domain()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    ses_tenant TEXT := trim(COALESCE(
        NEW.headers ->> 'X-Dugble-Internal-SES-Tenant',
        ''
    ));
    configuration_set TEXT := trim(COALESCE(
        NEW.headers ->> 'X-Dugble-Internal-SES-Configuration-Set',
        ''
    ));
    email_stream TEXT := trim(COALESCE(
        NEW.headers ->> 'X-Dugble-Internal-Email-Stream',
        ''
    ));
    sender_domain TEXT := lower(split_part(trim(NEW.from_email), '@', 2));
BEGIN
    -- dugble.me is provider-owned bootstrap infrastructure. It never belongs
    -- to a customer domain record or customer SES tenant.
    IF sender_domain = 'dugble.me' OR ses_tenant = 'dugble-sandbox' THEN
        IF lower(trim(NEW.from_email)) <> 'onboarding@dugble.me'
           OR NEW.sender_domain_id IS NOT NULL
           OR NEW.message_type <> 'transactional'
           OR ses_tenant <> 'dugble-sandbox'
           OR configuration_set <> 'dugble-transactional'
           OR email_stream <> 'transactional' THEN
            RAISE EXCEPTION 'invalid Dugble sandbox email route'
                USING ERRCODE = '23514';
        END IF;

        RETURN NEW;
    END IF;

    IF NEW.sender_domain_id IS NULL THEN
        RAISE EXCEPTION 'customer sender domain is required'
            USING ERRCODE = '23514';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM domains AS domain_record
        WHERE domain_record.id = NEW.sender_domain_id
          AND domain_record.team_id = NEW.team_id
          AND domain_record.provider = lower(trim(NEW.delivery_provider))
          AND domain_record.provider_region = lower(trim(NEW.provider_region))
          AND domain_record.status = 'verified'
          AND domain_record.disabled_at IS NULL
          AND domain_record.health_status <> 'degraded'
    ) THEN
        RAISE EXCEPTION 'customer sender domain is not verified and healthy'
            USING ERRCODE = '23514';
    END IF;

    PERFORM 1
    FROM email_tenants
    WHERE team_id = NEW.team_id
      AND provider = lower(trim(NEW.delivery_provider))
      AND region = lower(trim(NEW.provider_region))
      AND status = 'active';

    IF NOT FOUND THEN
        RAISE EXCEPTION 'active customer email tenant is required'
            USING ERRCODE = '23514';
    END IF;

    RETURN NEW;
END;
$$;
