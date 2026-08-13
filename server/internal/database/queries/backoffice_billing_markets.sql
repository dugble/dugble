-- name: BackofficeListBillingMarkets :many
SELECT m.code, m.currency, c.minor_unit, m.is_enabled FROM billing_markets m JOIN currencies c ON c.code=m.currency ORDER BY m.code LIMIT sqlc.arg(page_limit)::int OFFSET sqlc.arg(page_offset)::int;
-- name: BackofficeGetBillingMarket :one
SELECT m.code, m.currency, c.minor_unit, m.is_enabled FROM billing_markets m JOIN currencies c ON c.code=m.currency WHERE m.code=sqlc.arg(code);
-- name: BackofficeCreateBillingMarket :exec
INSERT INTO billing_markets (code,currency,is_enabled) VALUES (sqlc.arg(code),sqlc.arg(currency),sqlc.arg(is_enabled));
-- name: BackofficeSetBillingMarketEnabled :execrows
UPDATE billing_markets SET is_enabled=sqlc.arg(is_enabled) WHERE code=sqlc.arg(code);
