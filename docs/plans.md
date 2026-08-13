# Plans

Customer dashboard HTTP contracts for team billing. These are public,
team-scoped endpoints and not backoffice routes.

All requests require an authenticated session and the selected team's UUID in
the `X-Team-ID` header. State-changing browser requests also require the
`X-CSRF-Token` header described in [README.md](README.md).

Money values use integer minor units in the currency returned by the API. For
example, `amount_units: 1500` represents 15.00 when the currency has two decimal
places. Format values using the accompanying ISO currency code rather than
assuming a currency or decimal precision.

## `GET /plans`

- Session: required.
- Team header: required.
- CSRF: not required.

### Payload

No JSON request body.

### Response — `200 OK`

```json
{
  "success": true,
  "data": [
    {
      "code": "growth",
      "name": "Growth",
      "price": {
        "id": "string",
        "currency": "GHS",
        "amount_units": 1500
      },
      "available": true,
      "current": false,
      "pending": false,
      "effective_at": "2026-08-13T00:00:00Z"
    }
  ]
}
```

`price` is omitted when no effective price is available for the team's billing
market. `current` identifies the active plan, while `pending` identifies a plan
change scheduled for the next billing period.

### Errors

Errors use the standard envelope in [README.md](README.md). Relevant responses
include `401 Unauthorized`, `403 Forbidden`, and `503 Service Unavailable`.

