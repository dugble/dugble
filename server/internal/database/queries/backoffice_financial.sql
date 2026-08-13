-- name: BackofficeListPayments :many
SELECT payment.*, team.name AS team_name FROM payment_transactions AS payment JOIN teams AS team ON team.id=payment.team_id
WHERE (sqlc.narg(team_id)::uuid IS NULL OR payment.team_id=sqlc.narg(team_id)::uuid) AND (sqlc.arg(status)::text='' OR payment.status=sqlc.arg(status)::text) AND (sqlc.arg(provider)::text='' OR payment.provider=sqlc.arg(provider)::text)
ORDER BY payment.created_at DESC,payment.id DESC LIMIT sqlc.arg(page_limit)::int OFFSET sqlc.arg(page_offset)::int;
-- name: BackofficeGetPayment :one
SELECT payment.*,team.name AS team_name FROM payment_transactions payment JOIN teams team ON team.id=payment.team_id WHERE payment.id=sqlc.arg(id);
-- name: BackofficeListSubscriptions :many
SELECT subscription.*,team.name AS team_name,wallet.billing_market,wallet.currency FROM team_subscriptions subscription JOIN teams team ON team.id=subscription.team_id JOIN team_wallets wallet ON wallet.team_id=subscription.team_id
WHERE (sqlc.narg(team_id)::uuid IS NULL OR subscription.team_id=sqlc.narg(team_id)::uuid) AND (sqlc.arg(status)::text='' OR subscription.status=sqlc.arg(status)::text)
ORDER BY subscription.updated_at DESC,subscription.id DESC LIMIT sqlc.arg(page_limit)::int OFFSET sqlc.arg(page_offset)::int;
-- name: BackofficeGetSubscription :one
SELECT subscription.*,team.name AS team_name,wallet.billing_market,wallet.currency FROM team_subscriptions subscription JOIN teams team ON team.id=subscription.team_id JOIN team_wallets wallet ON wallet.team_id=subscription.team_id WHERE subscription.id=sqlc.arg(id);
-- name: BackofficeListSubscriptionCharges :many
SELECT * FROM subscription_charges WHERE subscription_id=sqlc.arg(subscription_id) ORDER BY created_at DESC,id DESC LIMIT sqlc.arg(page_limit)::int OFFSET sqlc.arg(page_offset)::int;
