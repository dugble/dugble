-- name: BackofficeListWallets :many
SELECT wallet.*, subscription.plan_code AS tier, team.name AS team_name
FROM team_wallets AS wallet
JOIN teams AS team ON team.id = wallet.team_id
JOIN team_subscriptions AS subscription ON subscription.team_id = wallet.team_id
ORDER BY wallet.updated_at DESC, wallet.team_id
LIMIT sqlc.arg(page_limit)
OFFSET sqlc.arg(page_offset);

-- name: BackofficeGetWallet :one
SELECT wallet.*, subscription.plan_code AS tier, team.name AS team_name
FROM team_wallets AS wallet
JOIN teams AS team ON team.id = wallet.team_id
JOIN team_subscriptions AS subscription ON subscription.team_id = wallet.team_id
WHERE wallet.team_id = sqlc.arg(team_id);

-- name: BackofficeListWalletTransactions :many
SELECT ledger.*, team.name AS team_name, wallet.currency
FROM wallet_ledger AS ledger
JOIN teams AS team ON team.id = ledger.team_id
JOIN team_wallets AS wallet ON wallet.team_id = ledger.team_id
WHERE sqlc.narg(team_id)::uuid IS NULL
   OR ledger.team_id = sqlc.narg(team_id)::uuid
ORDER BY ledger.created_at DESC, ledger.id DESC
LIMIT sqlc.arg(page_limit)
OFFSET sqlc.arg(page_offset);

-- name: BackofficeAdjustWallet :one
WITH locked_wallet AS MATERIALIZED (
    SELECT *
    FROM team_wallets AS wallet
    WHERE wallet.team_id = sqlc.arg(team_id)
    FOR UPDATE
),
inserted_ledger AS (
    INSERT INTO wallet_ledger (team_id, amount_units, transaction_type, reference_id)
    SELECT team_id, sqlc.arg(amount_units), 'adjustment', sqlc.arg(reference_id)
    FROM locked_wallet
    WHERE sqlc.arg(amount_units)::bigint <> 0
      AND balance_units::numeric + sqlc.arg(amount_units)::numeric
          BETWEEN 0 AND 9223372036854775807
    ON CONFLICT (team_id, transaction_type, reference_id) DO NOTHING
    RETURNING team_id, amount_units
)
UPDATE team_wallets AS wallet
SET balance_units = wallet.balance_units + ledger.amount_units,
    updated_at = now()
FROM inserted_ledger AS ledger
WHERE wallet.team_id = ledger.team_id
RETURNING wallet.*;

-- name: BackofficeGetWalletTransaction :one
SELECT ledger.*, team.name AS team_name, wallet.currency
FROM wallet_ledger AS ledger
JOIN teams AS team ON team.id = ledger.team_id
JOIN team_wallets AS wallet ON wallet.team_id = ledger.team_id
WHERE ledger.team_id = sqlc.arg(team_id) AND ledger.id = sqlc.arg(transaction_id);
