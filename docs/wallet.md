# Wallet

Customer dashboard HTTP contracts for team billing. These are public,
team-scoped endpoints and not backoffice routes.

All requests require an authenticated session and the selected team's UUID in
the `X-Team-ID` header. State-changing browser requests also require the
`X-CSRF-Token` header described in [README.md](README.md).

Money values use integer minor units in the currency returned by the API. For
example, `amount_units: 1500` represents 15.00 when the currency has two decimal
places. Format values using the accompanying ISO currency code rather than
assuming a currency or decimal precision.

## `GET /wallet`

- Session: required.
- Team header: required.
- CSRF: not required.

### Payload

No JSON request body.

### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "team_id": "string",
    "currency": "GHS",
    "balance_units": 25000,
    "created_at": "2026-08-01T00:00:00Z",
    "updated_at": "2026-08-13T00:00:00Z"
  }
}
```

### Errors

Errors use the standard envelope in [README.md](README.md). Relevant responses
include `401 Unauthorized`, `403 Forbidden`, and `404 Not Found`.

## `GET /wallet/ledger`

- Session: required.
- Team header: required.
- CSRF: not required.

### Query parameters

- `limit` — optional integer page size from 1 to 100; defaults to 50 when omitted.
- `offset` — optional non-negative integer page offset; defaults to 0 when omitted.

### Payload

No JSON request body.

### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "entries": [
      {
        "id": "string",
        "team_id": "string",
        "usage_authorization_id": "string",
        "subscription_charge_id": "string",
        "amount_units": -500,
        "transaction_type": "debit",
        "reference_id": "string",
        "created_at": "2026-08-13T00:00:00Z"
      }
    ],
    "limit": 50,
    "offset": 0
  }
}
```

`usage_authorization_id` and `subscription_charge_id` are omitted when they do
not apply to an entry. Credits have a positive `amount_units`; debits have a
negative value.

### Errors

An invalid `limit` or `offset` returns `400 Bad Request`. Other errors use the
standard envelope in [README.md](README.md).

## `POST /wallet/topup`

Creates a hosted checkout transaction. Redirect the customer to the returned
checkout URL; do not mark the wallet as funded until the API confirms payment.

- Session: required.
- Team header: required.
- CSRF: required for browser requests.

### Payload

```json
{
  "amount_units": 5000,
  "description": "Wallet top-up"
}
```

### Response — `201 Created`

```json
{
  "success": true,
  "data": {
    "transaction_id": "string",
    "client_reference": "string",
    "checkout_id": "string",
    "checkout_url": "https://example.com/checkout/string",
    "checkout_direct_url": "https://example.com/checkout/string/direct"
  }
}
```

### Errors

A non-positive amount returns `400 Bad Request`. A blank description defaults to
`Dugble wallet top-up`. Provider failures may
return `503 Service Unavailable`; other errors use the standard envelope in
[README.md](README.md).

## Provider callbacks

`POST /wallet/webhook/hubtel` is a payment-provider callback, not a customer
dashboard endpoint. Frontend applications must not call it or use it to infer
payment completion. Refresh `GET /wallet` and `GET /wallet/ledger` after the
hosted checkout completes to display the authoritative balance and transaction
history.
