-- name: CreatePaymentTransaction :one
INSERT INTO payment_transactions (
    team_id,
    provider,
    client_reference,
    currency,
    amount_units
)
SELECT
    sqlc.arg(team_id),
    sqlc.arg(provider),
    sqlc.arg(client_reference),
    sqlc.arg(currency),
    sqlc.arg(amount_units)
FROM team_wallets tw
WHERE tw.team_id = sqlc.arg(team_id)
  AND tw.currency = sqlc.arg(currency)
RETURNING *;

-- name: GetPaymentTransactionByID :one
SELECT *
FROM payment_transactions
WHERE id = sqlc.arg(id)
  AND team_id = sqlc.arg(team_id);

-- name: GetPaymentTransactionByClientReference :one
SELECT *
FROM payment_transactions
WHERE provider = sqlc.arg(provider)
  AND client_reference = sqlc.arg(client_reference);

-- name: ListPaymentTransactionsByTeam :many
SELECT *
FROM payment_transactions
WHERE team_id = sqlc.arg(team_id)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);

-- name: MarkPaymentTransactionPaid :one
UPDATE payment_transactions
SET status = 'paid',
    provider_transaction_id = sqlc.arg(provider_transaction_id),
    paid_at = now(),
    updated_at = now()
WHERE id = sqlc.arg(id)
  AND team_id = sqlc.arg(team_id)
  AND amount_units = sqlc.arg(amount_units)
  AND status = 'pending'
RETURNING *;
