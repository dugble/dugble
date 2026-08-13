-- name: BackofficeListCurrencies :many
SELECT code, minor_unit, is_enabled FROM currencies ORDER BY code LIMIT sqlc.arg(page_limit)::int OFFSET sqlc.arg(page_offset)::int;
-- name: BackofficeGetCurrency :one
SELECT code, minor_unit, is_enabled FROM currencies WHERE code = sqlc.arg(code);
-- name: BackofficeCreateCurrency :one
INSERT INTO currencies (code, minor_unit, is_enabled) VALUES (sqlc.arg(code), sqlc.arg(minor_unit), sqlc.arg(is_enabled)) RETURNING code, minor_unit, is_enabled;
-- name: BackofficeSetCurrencyEnabled :one
UPDATE currencies SET is_enabled = sqlc.arg(is_enabled) WHERE code = sqlc.arg(code) RETURNING code, minor_unit, is_enabled;
