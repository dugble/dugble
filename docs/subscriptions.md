# Subscriptions

Customer dashboard HTTP contracts for team billing. These are public,
team-scoped endpoints and not backoffice routes.

All requests require an authenticated session and the selected team's UUID in
the `X-Team-ID` header. State-changing browser requests also require the
`X-CSRF-Token` header described in [README.md](README.md).

Money values use integer minor units in the currency returned by the API. For
example, `amount_units: 1500` represents 15.00 when the currency has two decimal
places. Format values using the accompanying ISO currency code rather than
assuming a currency or decimal precision.

## `GET /subscription`

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
    "id": "string",
    "team_id": "string",
    "plan_code": "growth",
    "status": "active",
    "current_period_start": "2026-08-01T00:00:00Z",
    "current_period_end": "2026-09-01T00:00:00Z",
    "pending_plan_code": "scale",
    "pending_plan_effective_at": "2026-09-01T00:00:00Z",
    "cancel_at_period_end": false,
    "created_at": "2026-08-01T00:00:00Z",
    "updated_at": "2026-08-13T00:00:00Z"
  }
}
```

`pending_plan_code` and `pending_plan_effective_at` are omitted when no plan
change is scheduled.

### Errors

Errors use the standard envelope in [README.md](README.md). Relevant responses
include `401 Unauthorized`, `403 Forbidden`, and `404 Not Found`.

## `GET /subscription/charges`

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
    "charges": [
      {
        "id": "string",
        "subscription_id": "string",
        "plan_price_id": "string",
        "plan_code": "growth",
        "billing_market": "GH",
        "currency": "GHS",
        "period_start": "2026-08-01T00:00:00Z",
        "period_end": "2026-09-01T00:00:00Z",
        "amount_units": 1500,
        "status": "applied",
        "attempt_count": 1,
        "last_attempted_at": "2026-08-01T00:00:00Z",
        "applied_at": "2026-08-01T00:00:00Z",
        "reference_id": "string",
        "communication_credit": {
          "id": "string",
          "granted_units": 10000,
          "consumed_units": 250,
          "remaining_units": 9750
        },
        "created_at": "2026-08-01T00:00:00Z"
      }
    ],
    "limit": 20,
    "offset": 0
  }
}
```

`failure_code`, `applied_at`, and `communication_credit` are omitted when they
do not apply to the charge.

### Errors

An invalid `limit` or `offset` returns `400 Bad Request`. Other errors use the
standard envelope in [README.md](README.md).

## `POST /subscription`

Schedules a plan for the next billing period.

- Session: required.
- Team header: required.
- CSRF: required for browser requests.

### Payload

```json
{
  "plan": "growth"
}
```

Supported plan values are `growth`, `scale`, and `enterprise`.

### Response — `200 OK`

Returns the subscription object documented under `GET /subscription`.

### Errors

Invalid or unavailable plans return the standard error envelope. Relevant
responses include `400 Bad Request`, `403 Forbidden`, and `409 Conflict`.

## `POST /subscription/cancel-change`

Removes a pending plan change without cancelling the active subscription.

- Session: required.
- Team header: required.
- CSRF: required for browser requests.

### Payload

No JSON request body.

### Response — `200 OK`

Returns the updated subscription object documented under `GET /subscription`.

### Errors

Errors use the standard envelope in [README.md](README.md). A missing pending
change may return `409 Conflict`.

## `POST /subscription/cancel`

Schedules the active subscription to cancel at the end of its current period.

- Session: required.
- Team header: required.
- CSRF: required for browser requests.

### Payload

No JSON request body.

### Response — `200 OK`

Returns the updated subscription object with `cancel_at_period_end: true`.

### Errors

Errors use the standard envelope in [README.md](README.md). Relevant responses
include `403 Forbidden` and `409 Conflict`.

## `POST /subscription/reactivate`

Removes a scheduled end-of-period cancellation.

- Session: required.
- Team header: required.
- CSRF: required for browser requests.

### Payload

No JSON request body.

### Response — `200 OK`

Returns the updated subscription object with `cancel_at_period_end: false`.

### Errors

Errors use the standard envelope in [README.md](README.md). Relevant responses
include `403 Forbidden` and `409 Conflict`.

