ALTER TABLE domains
    DROP CONSTRAINT IF EXISTS chk_domains_capabilities,
    DROP COLUMN IF EXISTS open_tracking,
    DROP COLUMN IF EXISTS click_tracking,
    DROP COLUMN IF EXISTS tracking_subdomain,
    DROP COLUMN IF EXISTS active_tracking_subdomain,
    DROP COLUMN IF EXISTS sending_enabled,
    DROP COLUMN IF EXISTS receiving_enabled;

ALTER TABLE domain_claims
    DROP CONSTRAINT IF EXISTS chk_domain_claims_capabilities,
    DROP COLUMN IF EXISTS open_tracking,
    DROP COLUMN IF EXISTS click_tracking,
    DROP COLUMN IF EXISTS tracking_subdomain,
    DROP COLUMN IF EXISTS sending_enabled,
    DROP COLUMN IF EXISTS receiving_enabled;
