-- name: BackofficeListSMSRates :many
SELECT * FROM sms_rates ORDER BY effective_from DESC,created_at DESC LIMIT sqlc.arg(page_limit)::int OFFSET sqlc.arg(page_offset)::int;
-- name: BackofficeGetSMSRate :one
SELECT * FROM sms_rates WHERE id=sqlc.arg(id);
-- name: BackofficeCreateSMSRate :one
INSERT INTO sms_rates (destination_country,route_type,tier,currency,cost_units,effective_from,effective_until) VALUES (sqlc.arg(destination_country),sqlc.arg(route_type),sqlc.arg(tier),sqlc.arg(currency),sqlc.arg(cost_units),sqlc.arg(effective_from),sqlc.narg(effective_until)) RETURNING *;
-- name: BackofficeCloseSMSRate :one
UPDATE sms_rates SET effective_until=sqlc.arg(effective_until) WHERE id=sqlc.arg(id) AND sqlc.arg(effective_until)>effective_from AND (effective_until IS NULL OR sqlc.arg(effective_until)<effective_until) RETURNING *;
-- name: BackofficeReplaceSMSRate :one
WITH closed AS (
  UPDATE sms_rates AS current SET effective_until=sqlc.arg(effective_from)
  WHERE current.id=sqlc.arg(target_id) AND sqlc.arg(effective_from)>current.effective_from AND (current.effective_until IS NULL OR sqlc.arg(effective_from)<current.effective_until)
  RETURNING destination_country,route_type,tier,currency
)
INSERT INTO sms_rates (destination_country,route_type,tier,currency,cost_units,effective_from,effective_until)
SELECT destination_country,route_type,tier,currency,sqlc.arg(cost_units),sqlc.arg(effective_from),sqlc.narg(effective_until) FROM closed
RETURNING *;
-- name: BackofficeListProductRates :many
SELECT * FROM product_rates ORDER BY effective_from DESC,created_at DESC LIMIT sqlc.arg(page_limit)::int OFFSET sqlc.arg(page_offset)::int;
-- name: BackofficeGetProductRate :one
SELECT * FROM product_rates WHERE id=sqlc.arg(id);
-- name: BackofficeCreateProductRate :one
INSERT INTO product_rates (product,meter,billing_market,tier,currency,cost_units,effective_from,effective_until) VALUES (sqlc.arg(product),sqlc.arg(meter),sqlc.arg(billing_market),sqlc.arg(tier),sqlc.arg(currency),sqlc.arg(cost_units),sqlc.arg(effective_from),sqlc.narg(effective_until)) RETURNING *;
-- name: BackofficeCloseProductRate :one
UPDATE product_rates SET effective_until=sqlc.arg(effective_until) WHERE id=sqlc.arg(id) AND sqlc.arg(effective_until)>effective_from AND (effective_until IS NULL OR sqlc.arg(effective_until)<effective_until) RETURNING *;
-- name: BackofficeReplaceProductRate :one
WITH closed AS (
  UPDATE product_rates AS current SET effective_until=sqlc.arg(effective_from)
  WHERE current.id=sqlc.arg(target_id) AND sqlc.arg(effective_from)>current.effective_from AND (current.effective_until IS NULL OR sqlc.arg(effective_from)<current.effective_until)
  RETURNING product,meter,billing_market,tier,currency
)
INSERT INTO product_rates (product,meter,billing_market,tier,currency,cost_units,effective_from,effective_until)
SELECT product,meter,billing_market,tier,currency,sqlc.arg(cost_units),sqlc.arg(effective_from),sqlc.narg(effective_until) FROM closed
RETURNING *;
-- name: BackofficeListFXRates :many
SELECT *
FROM fx_rates
ORDER BY effective_from DESC, created_at DESC
LIMIT sqlc.arg(page_limit)::int
OFFSET sqlc.arg(page_offset)::int;


-- name: BackofficeGetFXRate :one
SELECT *
FROM fx_rates
WHERE id = sqlc.arg(id);

-- name: BackofficeCloseFXRate :one
UPDATE fx_rates
SET effective_until = sqlc.arg(effective_until)
WHERE id = sqlc.arg(id)
  AND sqlc.arg(effective_until) > effective_from
  AND (effective_until IS NULL OR sqlc.arg(effective_until) < effective_until)
RETURNING *;


-- name: BackofficeCreateFXRate :one
INSERT INTO fx_rates (
    base_currency,
    quote_currency,
    rate,
    effective_from,
    effective_until
)
VALUES (
    upper(sqlc.arg(base_currency))::char(3),
    upper(sqlc.arg(quote_currency))::char(3),
    sqlc.arg(rate),
    sqlc.arg(effective_from),
    sqlc.narg(effective_until)
)
RETURNING *;


-- name: BackofficeReplaceFXRate :one
WITH closed_rate AS (
    UPDATE fx_rates AS current
    SET effective_until = sqlc.arg(effective_from)
    WHERE current.id = sqlc.arg(target_id)
      AND current.effective_from < sqlc.arg(effective_from)
      AND (
          current.effective_until IS NULL
          OR current.effective_until > sqlc.arg(effective_from)
      )
    RETURNING base_currency, quote_currency
)
INSERT INTO fx_rates (
    base_currency,
    quote_currency,
    rate,
    effective_from
)
SELECT
    base_currency,
    quote_currency,
    sqlc.arg(rate),
    sqlc.arg(effective_from)
FROM closed_rate
RETURNING *;

-- name: BackofficeListAllowancePolicies :many
SELECT * FROM allowance_policies ORDER BY effective_from DESC,created_at DESC LIMIT sqlc.arg(page_limit)::int OFFSET sqlc.arg(page_offset)::int;
-- name: BackofficeGetAllowancePolicy :one
SELECT * FROM allowance_policies WHERE id=sqlc.arg(id);
-- name: BackofficeCreateAllowancePolicy :one
INSERT INTO allowance_policies (product,meter,billing_market,tier,included_quantity,cadence,effective_from,effective_until) VALUES (sqlc.arg(product),sqlc.arg(meter),sqlc.arg(billing_market),sqlc.arg(tier),sqlc.arg(included_quantity),sqlc.arg(cadence),sqlc.arg(effective_from),sqlc.narg(effective_until)) RETURNING *;
-- name: BackofficeCloseAllowancePolicy :one
UPDATE allowance_policies SET effective_until=sqlc.arg(effective_until),updated_at=now() WHERE id=sqlc.arg(id) AND sqlc.arg(effective_until)>effective_from AND (effective_until IS NULL OR sqlc.arg(effective_until)<effective_until) RETURNING *;
-- name: BackofficeReplaceAllowancePolicy :one
WITH closed AS (
  UPDATE allowance_policies AS current SET effective_until=sqlc.arg(effective_from),updated_at=now()
  WHERE current.id=sqlc.arg(target_id) AND sqlc.arg(effective_from)>current.effective_from AND (current.effective_until IS NULL OR sqlc.arg(effective_from)<current.effective_until)
  RETURNING product,meter,billing_market,tier,cadence
)
INSERT INTO allowance_policies (product,meter,billing_market,tier,included_quantity,cadence,effective_from,effective_until)
SELECT product,meter,billing_market,tier,sqlc.arg(included_quantity),cadence,sqlc.arg(effective_from),sqlc.narg(effective_until) FROM closed
RETURNING *;
