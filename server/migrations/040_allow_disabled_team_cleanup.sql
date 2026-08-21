-- Allow an owner to repeat DELETE on an already-disabled team.
-- The normal DELETE remains a soft delete. A repeated DELETE on a disabled
-- team clears its memberships so the user can finish account deletion while
-- retaining the disabled team record for history/audit purposes.

CREATE OR REPLACE FUNCTION clear_disabled_team_memberships()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF OLD.status = 'disabled' AND NEW.status = 'disabled' THEN
        DELETE FROM team_members
        WHERE team_id = NEW.id;
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_clear_disabled_team_memberships ON teams;

CREATE TRIGGER trg_clear_disabled_team_memberships
AFTER UPDATE OF status ON teams
FOR EACH ROW
WHEN (OLD.status = 'disabled' AND NEW.status = 'disabled')
EXECUTE FUNCTION clear_disabled_team_memberships();
