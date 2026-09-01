# SMS API

Dashboard-facing HTTP contract for the SMS API.

The routes and payloads in this document are based on `server/internal/messaging/sms`. Request and response shapes below reflect the public Go types used by the handlers, not internal database-only fields.

## Conventions

- Read endpoints require the SMS read permission.
- Send, update, cancel, and status-sync endpoints require the SMS send permission.
- `POST /sms` and `POST /sms/batch` require an `Idempotency-Key` header. The key must be present and at most 256 characters.
- Collection pagination uses `limit` and `offset`.
- `GET /sms` defaults `limit` to 50 when it is omitted, zero, or greater than 100. Negative offsets are treated as 0.
- `GET /sms/:message_id/events` defaults `limit` to 100 when it is omitted, zero, or greater than 100.
- `POST /sms` and `POST /sms/batch` return `202 Accepted`.
- Timestamps are RFC 3339/ISO-8601 strings in JSON.

Successful responses use the repository's standard envelope:

```json
{
  "success": true,
  "data": {}
}
```

The `data` value is replaced below with the endpoint's actual response shape.

---

## List SMS messages

### `GET /sms`

Returns SMS messages for the current team. Each item is the public `SMSResponse` shape described in [Get an SMS](#get-an-sms).

#### Query parameters

| Parameter | Type | Description |
| --- | --- | --- |
| `limit` | integer | Maximum number of results. Values `<= 0` or `> 100` use the default of 50. |
| `offset` | integer | Number of results to skip. Negative values are treated as 0. |
| `status` | string | Exact SMS status. Must be one of the [documented SMS statuses](#sms-statuses). |
| `sender` | string | Exact sender identity, matching the response's `from` value. |
| `start_date` | RFC 3339 timestamp | Include messages created at or after this timestamp. |
| `end_date` | RFC 3339 timestamp | Include messages created at or before this timestamp. Must not precede `start_date`. |
| `search` | string | Case-insensitive partial match against recipient, sender, message body, or provider message ID. |

All supplied filters are combined with AND. Pagination is applied after filtering.

#### Request body

None.

#### Response

`200 OK`.

```json
{
  "success": true,
  "data": [
    {
      "object": "sms",
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "message_id": "provider-message-id",
      "to": "+233200000000",
      "from": "+233500000000",
      "body": "Hello from Dugble",
      "last_event": "delivered",
      "destination": {
        "country": "GH"
      },
      "segments": 1,
      "metadata": {
        "contact_id": "contact-123"
      },
      "scheduled_at": "2026-08-25T09:00:00Z",
      "submitted_at": "2026-08-25T09:00:02Z",
      "delivered_at": "2026-08-25T09:00:05Z",
      "created_at": "2026-08-25T08:59:58Z",
      "updated_at": "2026-08-25T09:00:05Z"
    }
  ]
}
```

`message_id`, `scheduled_at`, `submitted_at`, and `delivered_at` are nullable in the underlying response type. `failure` is only present for `undelivered`, `rejected`, `failed`, or `expired` messages.

---

## Send an SMS

### `POST /sms`

Queues one SMS for delivery. The message may be sent immediately or scheduled for a future time.

**Required header:** `Idempotency-Key`

#### Request body

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `to` | string | Yes | Destination phone number. Must be a valid E.164 number. |
| `from` | string | Yes | Sender ID. Must be approved for the current team and be at most 11 characters. |
| `body` | string | Yes | SMS body. Must not be blank and may contain at most 1,600 characters. |
| `metadata` | JSON value | No | Application metadata. Defaults to `{}` when omitted. Must be valid JSON. |
| `scheduled_at` | string | No | Future delivery time in RFC 3339/ISO-8601 format, or a relative value such as `in 5 min`. Must be at least 30 seconds in the future. |

```json
{
  "to": "+233200000000",
  "from": "+233500000000",
  "body": "Hello from Dugble",
  "metadata": {
    "contact_id": "contact-123"
  },
  "scheduled_at": "2026-08-25T09:00:00Z"
}
```

The server derives the destination country from `to`; it is not a request field.

#### Response

`202 Accepted` with a `Location` header pointing to `/sms/:message_id`.

```json
{
  "success": true,
  "data": {
    "object": "sms",
    "id": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

---

## Send SMS in bulk

### `POST /sms/batch`

Queues multiple SMS messages. A batch must contain at least one message and may contain at most 50 messages.

**Required header:** `Idempotency-Key`

#### Request body

The API accepts either a top-level array or an object containing `messages`.

Object form:

```json
{
  "messages": [
    {
      "to": "+233200000000",
      "from": "+233500000000",
      "body": "Hello from Dugble",
      "metadata": {
        "contact_id": "contact-123"
      }
    }
  ]
}
```

Each item uses the same fields and validation rules as `POST /sms`.

#### Response

`202 Accepted`.

```json
{
  "success": true,
  "data": [
    {
      "object": "sms",
      "id": "550e8400-e29b-41d4-a716-446655440000"
    },
    {
      "object": "sms",
      "id": "650e8400-e29b-41d4-a716-446655440000"
    }
  ]
}
```

---

## Get an SMS

### `GET /sms/:message_id`

Returns the complete public representation of an SMS message.

#### Path parameters

| Parameter | Type | Description |
| --- | --- | --- |
| `message_id` | UUID string | SMS message ID. |

#### Request body

None.

#### Response

`200 OK`.

```json
{
  "success": true,
  "data": {
    "object": "sms",
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "message_id": "provider-message-id",
    "to": "+233200000000",
    "from": "+233500000000",
    "body": "Hello from Dugble",
    "last_event": "delivered",
    "destination": {
      "country": "GH"
    },
    "segments": 1,
    "metadata": {
      "contact_id": "contact-123"
    },
    "scheduled_at": "2026-08-25T09:00:00Z",
    "submitted_at": "2026-08-25T09:00:02Z",
    "delivered_at": "2026-08-25T09:00:05Z",
    "created_at": "2026-08-25T08:59:58Z",
    "updated_at": "2026-08-25T09:00:05Z"
  }
}
```

For failed delivery states, the response includes `failure`:

```json
"failure": {
  "code": "SMS_FAILED",
  "message": "SMS delivery failed"
}
```

The public failure codes are `SMS_UNDELIVERED`, `SMS_REJECTED`, `SMS_FAILED`, and `SMS_EXPIRED`.

---

## Update an SMS

### `PATCH /sms/:message_id`

Reschedules an eligible pending SMS. Only pending scheduled messages outside the delivery cutoff can be updated.

#### Path parameters

| Parameter | Type | Description |
| --- | --- | --- |
| `message_id` | UUID string | SMS message ID. |

#### Request body

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `scheduled_at` | string | Yes | Future RFC 3339/ISO-8601 time. Must be at least 30 seconds in the future. |

```json
{
  "scheduled_at": "2026-08-25T09:30:00Z"
}
```

Unlike `POST /sms`, this endpoint requires an ISO-8601 timestamp; relative values such as `in 5 min` are not accepted here.

#### Response

`200 OK`.

```json
{
  "success": true,
  "data": {
    "object": "sms",
    "id": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

---

## Cancel an SMS

### `POST /sms/:message_id/cancel`

Cancels an eligible pending scheduled SMS. Only pending scheduled messages outside the delivery cutoff can be canceled.

#### Path parameters

| Parameter | Type | Description |
| --- | --- | --- |
| `message_id` | UUID string | SMS message ID. |

#### Request body

None.

#### Response

`200 OK`.

```json
{
  "success": true,
  "data": {
    "object": "sms",
    "id": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

---

## Sync SMS status

### `POST /sms/:message_id/sync-status`

Requests a fresh provider status for an SMS that has already been submitted to a provider.

#### Path parameters

| Parameter | Type | Description |
| --- | --- | --- |
| `message_id` | UUID string | SMS message ID. |

#### Request body

None.

#### Response

`200 OK` with the same `SMSResponse` shape as `GET /sms/:message_id`.

```json
{
  "success": true,
  "data": {
    "object": "sms",
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "message_id": "provider-message-id",
    "to": "+233200000000",
    "from": "+233500000000",
    "body": "Hello from Dugble",
    "last_event": "delivered",
    "destination": {
      "country": "GH"
    },
    "segments": 1,
    "metadata": {},
    "scheduled_at": null,
    "submitted_at": "2026-08-25T09:00:02Z",
    "delivered_at": "2026-08-25T09:00:05Z",
    "created_at": "2026-08-25T08:59:58Z",
    "updated_at": "2026-08-25T09:00:05Z"
  }
}
```

The endpoint returns `400 Bad Request` when the SMS has not yet been submitted to a provider.

---

## List SMS events

### `GET /sms/:message_id/events`

Returns delivery events recorded for an SMS.

#### Path parameters

| Parameter | Type | Description |
| --- | --- | --- |
| `message_id` | UUID string | SMS message ID. |

#### Query parameters

| Parameter | Type | Description |
| --- | --- | --- |
| `limit` | integer | Maximum number of events. Values `<= 0` or `> 100` use the default of 100. |

#### Request body

None.

#### Response

`200 OK`.

```json
{
  "success": true,
  "data": {
    "object": "list",
    "data": [
      {
        "id": "event-id",
        "type": "delivered",
        "occurred_at": "2026-08-25T09:00:05Z",
        "provider": "provider-name",
        "code": "250",
        "message": "Delivered"
      }
    ]
  }
}
```

`provider`, `code`, and `message` are omitted when unavailable.

---

## SMS analytics

### `GET /sms/analytics`

Returns SMS delivery analytics for the current team.

#### Request body

None.

#### Response

`200 OK`.

```json
{
  "success": true,
  "data": {
    "object": "analytics",
    "windows": [
      {
        "days": 7,
        "rates": [
          {
            "name": "delivery_rate",
            "value": 0.98
          }
        ],
        "series": [
          {
            "date": "2026-08-25",
            "total": 100,
            "delivered": 98,
            "failed": 2
          }
        ]
      }
    ],
    "delivery_by_country": [
      {
        "country": "GH",
        "total": 100,
        "delivered": 98,
        "failed": 2
      }
    ]
  }
}
```

### Analytics fields

#### `windows[]`

| Field | Type | Description |
| --- | --- | --- |
| `days` | integer | Number of days represented by the analytics window. |
| `rates` | array | Named aggregate rates. |
| `rates[].name` | string | Rate name. |
| `rates[].value` | number | Rate value. |
| `series` | array | Daily analytics points. |
| `series[].date` | string | Date represented by the point. |
| `series[].total` | integer | Total messages. |
| `series[].delivered` | integer | Delivered messages. |
| `series[].failed` | integer | Failed messages. |

#### `delivery_by_country[]`

| Field | Type | Description |
| --- | --- | --- |
| `country` | string | Destination country. |
| `total` | integer | Total messages. |
| `delivered` | integer | Delivered messages. |
| `failed` | integer | Failed messages. |

---

## SMS statuses

The implementation defines these statuses:

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

For `undelivered`, `rejected`, `failed`, and `expired`, the public response includes a generated `failure` object with the corresponding failure code.

| Status | Failure code | Failure message |
| --- | --- | --- |
| `undelivered` | `SMS_UNDELIVERED` | `SMS could not be delivered` |
| `rejected` | `SMS_REJECTED` | `SMS was rejected` |
| `failed` | `SMS_FAILED` | `SMS delivery failed` |
| `expired` | `SMS_EXPIRED` | `SMS delivery expired` |

---

## SMS body encoding and segments

The server calculates SMS segments from the message body:

- GSM-7: up to 160 septets for one segment and 153 septets per segment for multipart messages.
- UCS-2: up to 70 characters for one segment and 67 characters per segment for multipart messages.
- The body is limited to 1,600 characters.

The public message response exposes the calculated `segments` count. The API does not currently expose the encoding/character analysis as a separate HTTP endpoint in the SMS route registration.

## Scheduling behavior

SMS scheduling is one-time scheduling. The API exposes a single `scheduled_at` timestamp; there is no recurring SMS schedule contract in this module.

A newly sent SMS can be scheduled at least 30 seconds in the future. Scheduled messages can be rescheduled or canceled only while they remain eligible and outside the delivery cutoff. The update endpoint uses a 15-second delivery cutoff.
