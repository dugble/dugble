-- name: CreateTeamWithOwner :one
WITH created_team AS (
    INSERT INTO teams (
        name,
        market_code,
        phone,
        address,
        website,
        created_by
    )
    VALUES (
        sqlc.arg(name),
        sqlc.arg(market_code),
        sqlc.arg(phone),
        sqlc.arg(address),
        sqlc.narg(website),
        sqlc.arg(owner_id)
    )
    RETURNING *
),
created_owner AS (
    INSERT INTO team_members (
        team_id,
        user_id,
        role,
        status
    )
    SELECT
        id,
        sqlc.arg(owner_id),
        'owner',
        'active'
    FROM created_team
    RETURNING team_id
),
created_wallet AS (
    INSERT INTO team_wallets (
        team_id,
        billing_market,
        currency
    )
    SELECT
        team.id,
        team.market_code,
        market.currency
    FROM created_team AS team
    JOIN billing_markets AS market
      ON market.code = team.market_code
     AND market.is_enabled = true
    RETURNING team_id
),
created_subscription AS (
    INSERT INTO team_subscriptions (
        team_id,
        plan_code,
        current_period_start,
        current_period_end
    )
    SELECT
        wallet.team_id,
        'growth',
        date_trunc('month', now() AT TIME ZONE 'UTC') AT TIME ZONE 'UTC',
        (
            date_trunc('month', now() AT TIME ZONE 'UTC')
            + interval '1 month'
        ) AT TIME ZONE 'UTC'
    FROM created_wallet AS wallet
    RETURNING team_id
)
SELECT created_team.*
FROM created_team
JOIN created_owner
    ON created_owner.team_id = created_team.id
JOIN created_wallet
    ON created_wallet.team_id = created_team.id
JOIN created_subscription
    ON created_subscription.team_id = created_team.id;

-- name: GetTeam :one
SELECT *
FROM teams
WHERE id = sqlc.arg(id);

-- name: ListTeamsForUser :many
SELECT t.*
FROM teams t
JOIN team_members tm ON tm.team_id = t.id
WHERE tm.user_id = sqlc.arg(user_id)
  AND tm.status = 'active'
  AND t.status = 'active'
ORDER BY t.created_at DESC;

-- name: UpdateTeam :one
UPDATE teams
SET name = sqlc.arg(name),
    updated_at = now()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DisableTeam :one
WITH current_team AS (
    SELECT status
    FROM teams
    WHERE id = sqlc.arg(id)
),
disabled_team AS (
    UPDATE teams
    SET status = 'disabled',
        updated_at = now()
    WHERE id = sqlc.arg(id)
      AND status = 'active'
    RETURNING *
),
cleared_members AS (
    DELETE FROM team_members
    WHERE team_id = sqlc.arg(id)
      AND (SELECT status FROM current_team) = 'disabled'
    RETURNING team_id
)
SELECT *
FROM disabled_team
UNION ALL
SELECT teams.*
FROM teams
WHERE teams.id = sqlc.arg(id)
  AND teams.status = 'disabled'
  AND NOT EXISTS (SELECT 1 FROM disabled_team);

-- name: CreateTeamMember :one
INSERT INTO team_members (
    team_id,
    user_id,
    role,
    status
) VALUES (
    sqlc.arg(team_id),
    sqlc.arg(user_id),
    sqlc.arg(role),
    sqlc.arg(status)
RETURNING *;

-- name: GetTeamMember :one
SELECT *
FROM team_members
WHERE team_id = sqlc.arg(team_id)
  AND user_id = sqlc.arg(user_id);

-- name: ListTeamMembers :many
SELECT *
FROM team_members
WHERE team_id = sqlc.arg(team_id)
ORDER BY created_at ASC;

-- name: ListActiveTeamOwnerRecipients :many
SELECT users.name, users.email, teams.name AS team_name
FROM team_members
JOIN users ON users.id = team_members.user_id
JOIN teams ON teams.id = team_members.team_id
WHERE team_members.team_id = sqlc.arg(team_id)
  AND team_members.role = 'owner'
  AND team_members.status = 'active'
ORDER BY users.id;

-- name: UpdateTeamMemberRole :one
UPDATE team_members
SET role = sqlc.arg(role),
    updated_at = now()
WHERE team_id = sqlc.arg(team_id)
  AND user_id = sqlc.arg(user_id)
RETURNING *;

-- name: RemoveTeamMember :exec
DELETE FROM team_members
WHERE team_id = sqlc.arg(team_id)
  AND user_id = sqlc.arg(user_id);
