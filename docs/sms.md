# SMS API

Dashboard-facing HTTP contract for the SMS API.

The routes and payloads in this document are based on `server/internal/modules/sms`.

## Conventions

- Read endpoints require the SMS read permission.
- Send, update, cancel, and status-sync endpoints require the SMS send permission.
- `POST /sms` and `POST /sms/batch` require an `Idempotency-Key` header.
- Collection pagination uses `limit` and `offset`.
- `POST /sms` and `POST /sms/batch` return `202 Accepted`.

---

## List SMS messages

### `GET /sms`

Returns SMS message summaries.

#### Query parameters

| Parameter | Type | Description |
| --- | --- | --- |
| `limit` | integer | Maximum number of results. |
| `offset` | integer | Number of results to skip. |

> **Current limitation:** the server currently implements only `limit` and `offset` for this endpoint. `status`, sender, date-range, and search filters are not yet supported server-side.

---

## Send an SMS

### `POST /sms`

Queues one SMS for delivery.

**Required header:** `Idempotency-Key`

#### Request body

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `to` | string | Yes | Destination phone number. |
| `from` | string | Yes | Sender value. |
| `body` | string | Yes | SMS message body. |
| `metadata` | object | No | Application metadata. |
| `scheduled_at` | string | No | Scheduled delivery time. |

```json
{
  "to": "+233200000000",
  "from": "+233500000000",
  "body": "Hello from Dugble",
  "metadata": {
    "contact_id": "string"
  }
}
```

Returns `202 Accepted` with a `Location` header pointing to `/sms/:message_id`.

---

## Send SMS in bulk

### `POST /sms/batch`

Queues multiple SMS messages. The request accepts either a top-level array or an object containing `messages`.

**Required header:** `Idempotency-Key`

```json
{
  "messages": [
    {
      "to": "+233200000000",
      "from": "+233500000000",
      "body": "Hello from Dugble"
    }
  ]
}
```

Returns `202 Accepted`.

---

## Get an SMS

### `GET /sms/:message_id`

Returns the complete public representation of an SMS message, including provider message ID, destination, segments, metadata, scheduling, failure information, and timestamps.

---

## Update an SMS

### `PATCH /sms/:message_id`

Updates an eligible SMS. The current request model supports scheduling.

```json
{
  "scheduled_at": "2026-08-25T09:00:00Z"
}
```

---

## Cancel an SMS

### `POST /sms/:message_id/cancel`

Cancels an eligible SMS. No request body.

---

## Sync SMS status

### `POST /sms/:message_id/sync-status`

Requests status synchronization with the provider. No request body.

---

## List SMS events

### `GET /sms/:message_id/events`

Returns delivery events recorded for an SMS.

#### Query parameters

| Parameter | Type | Description |
| --- | --- | --- |
| `limit` | integer | Maximum number of events to return. |

---

## SMS analytics

### `GET /sms/analytics`

Returns SMS delivery analytics, including delivery windows and delivery-by-country data.

---

## SMS statuses

The implementation defines:

- `queued`
- `processing`
- `submitted`
- `sent`
- `delivered`
- `undelivered`
- `rejected`
- `failed`
- `expired`
- `unknown`
- `canceled`

For failed delivery states, the public response may include a `failure` object.

| Status | Failure code |
| --- | --- |
| `undelivered` | `SMS_UNDELIVERED` |
| `rejected` | `SMS_REJECTED` |
| `failed` | `SMS_FAILED` |
| `expired` | `SMS_EXPIRED` |
