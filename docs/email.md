# Email API

Dashboard-facing HTTP contract for the Email API.

The schemas below are derived from the public Email module request/response types and handlers. They describe the fields exposed by the HTTP API; internal database fields that are not returned by the handler are intentionally omitted.

## Conventions

- `POST /emails` and `POST /emails/batch` require an `Idempotency-Key` header. The value must be at most 256 characters.
- Successful responses use the repository's standard JSON envelope. The `data` value is the endpoint's actual response shape shown below.
- Collection endpoints use `limit` and `offset` for pagination.
- Email addresses in request bodies may be supplied as an object or as a string such as `user@example.com` or `User <user@example.com>`.
- Timestamps are represented as RFC 3339/ISO-8601 strings in JSON.

## Email address formats

Object form:

```json
{
  "email": "user@example.com",
  "name": "User"
}
```

String forms include:

```text
user@example.com
User <user@example.com>
```

## Response shapes

### Send / mutation response

Used by `POST /emails`, each item from `POST /emails/batch`, `PATCH /emails/:message_id`, and `POST /emails/:message_id/cancel`.

```json
{
  "object": "email",
  "id": "550e8400-e29b-41d4-a716-446655440000"
}
```

### Email summary

Items returned by `GET /emails` have this shape:

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "to_email": "recipient@example.com",
  "to_name": "Recipient",
  "subject": "Welcome",
  "status": "queued",
  "provider": "ses",
  "queued_at": "2026-08-25T09:00:00Z",
  "submitted_at": "2026-08-25T09:00:02Z",
  "delivered_at": "2026-08-25T09:00:05Z",
  "created_at": "2026-08-25T08:59:58Z"
}
```

`to_name`, `provider`, `submitted_at`, and `delivered_at` are omitted when unavailable.

### Email retrieval response

`GET /emails/:message_id` returns:

```json
{
  "object": "email",
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "message_id": "provider-message-id",
  "to": ["Recipient <recipient@example.com>"],
  "from": "Sender <sender@example.com>",
  "created_at": "2026-08-25T08:59:58Z",
  "subject": "Welcome",
  "html": "<p>Hello</p>",
  "text": "Hello",
  "bcc": [],
  "cc": [],
  "reply_to": ["reply@example.com"],
  "last_event": "delivered",
  "scheduled_at": "2026-08-25T09:00:00Z",
  "tags": [{"name": "campaign", "value": "welcome"}]
}
```

`message_id` and `scheduled_at` may be `null`; `html` and `text` may be `null`.

### Email event

Items returned by `GET /emails/:message_id/events` have this shape:

```json
{
  "id": "event-id",
  "type": "delivered",
  "occurred_at": "2026-08-25T09:00:05Z",
  "provider": "ses",
  "code": "250",
  "message": "Delivered"
}
```

`provider`, `code`, and `message` are omitted when unavailable.

---

## List emails

### `GET /emails`

Returns email message summaries for the current tenant.

#### Query parameters

| Parameter | Type | Description |
| --- | --- | --- |
| `limit` | integer | Maximum number of results. |
| `offset` | integer | Number of results to skip. |

#### Request body

None.

#### Response

`200 OK`.

```json
{
  "success": true,
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "to_email": "recipient@example.com",
      "to_name": "Recipient",
      "subject": "Welcome",
      "status": "queued",
      "provider": "ses",
      "queued_at": "2026-08-25T09:00:00Z",
      "submitted_at": "2026-08-25T09:00:02Z",
      "delivered_at": "2026-08-25T09:00:05Z",
      "created_at": "2026-08-25T08:59:58Z"
    }
  ]
}
```

---

## Send an email

### `POST /emails`

Creates and queues a single transactional email for delivery.

**Required header:** `Idempotency-Key`

#### Request body

```json
{
  "from": {"email": "sender@example.com", "name": "Sender"},
  "reply_to": "reply@example.com",
  "to": ["Recipient <recipient@example.com>"],
  "cc": ["cc@example.com"],
  "bcc": ["bcc@example.com"],
  "subject": "Welcome",
  "html": "<p>Hello</p>",
  "text": "Hello",
  "headers": {"X-Custom-Header": "value"},
  "attachments": [
    {
      "content": "base64-or-content-string",
      "filename": "welcome.pdf",
      "path": "",
      "content_type": "application/pdf",
      "content_id": ""
    }
  ],
  "tags": [{"name": "campaign", "value": "welcome"}],
  "scheduled_at": "2026-08-25T09:00:00Z",
  "metadata": {"contact_id": "contact-id"}
}
```

`to` and `subject` are required. `from` is optional because the service can resolve the configured sender. `reply_to`, `cc`, and `bcc` accept one address or an array. `html` and `text`, headers, attachments, tags, scheduled time, and metadata are optional.

Returns `202 Accepted` and a `Location` header pointing to `/emails/:message_id`.

#### Response

```json
{
  "success": true,
  "data": {
    "object": "email",
    "id": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

---

## Send emails in bulk

### `POST /emails/batch`

Queues multiple transactional emails for delivery. The request accepts either a top-level array or an object containing `messages`.

**Required header:** `Idempotency-Key`

#### Request body

```json
{
  "messages": [
    {
      "to": "recipient@example.com",
      "subject": "Hello",
      "html": "<p>Hello</p>"
    },
    {
      "to": ["another@example.com"],
      "subject": "Second message",
      "text": "Hello again",
      "scheduled_at": "2026-08-25T09:00:00Z"
    }
  ]
}
```

The equivalent top-level array form is also accepted. Each item uses the same fields as `POST /emails`. The service limits a batch to 50 messages.

#### Response

`202 Accepted`.

```json
{
  "success": true,
  "data": [
    {"object": "email", "id": "550e8400-e29b-41d4-a716-446655440000"},
    {"object": "email", "id": "650e8400-e29b-41d4-a716-446655440000"}
  ]
}
```

---

## Get an email

### `GET /emails/:message_id`

Returns the public representation of an email message, including recipients, content, provider message ID, status through `last_event`, timestamps, scheduled time, and tags.

#### Request body

None.

#### Response

`200 OK`.

```json
{
  "success": true,
  "data": {
    "object": "email",
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "message_id": "provider-message-id",
    "to": ["Recipient <recipient@example.com>"],
    "from": "Sender <sender@example.com>",
    "created_at": "2026-08-25T08:59:58Z",
    "subject": "Welcome",
    "html": "<p>Hello</p>",
    "text": "Hello",
    "bcc": [],
    "cc": [],
    "reply_to": [],
    "last_event": "delivered",
    "scheduled_at": "2026-08-25T09:00:00Z",
    "tags": []
  }
}
```

---

## Update an email

### `PATCH /emails/:message_id`

Updates the schedule of an eligible pending email. Only pending scheduled emails can be updated.

#### Request body

```json
{
  "scheduled_at": "2026-08-25T10:00:00Z"
}
```

#### Response

`200 OK`.

```json
{
  "success": true,
  "data": {
    "object": "email",
    "id": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

---

## Cancel an email

### `POST /emails/:message_id/cancel`

Cancels an eligible pending scheduled email.

#### Request body

None.

#### Response

`200 OK`.

```json
{
  "success": true,
  "data": {
    "object": "email",
    "id": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

---

## List email events

### `GET /emails/:message_id/events`

Returns delivery events recorded for an email.

#### Query parameters

| Parameter | Type | Description |
| --- | --- | --- |
| `limit` | integer | Maximum number of events. |
| `offset` | integer | Number of events to skip. |

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
        "provider": "ses",
        "code": "250",
        "message": "Delivered"
      }
    ]
  }
}
```

---

## Email analytics

### `GET /emails/analytics`

Returns team-wide email delivery, open, click, and bounce analytics for 7-day, 30-day, and 90-day windows. Each window includes aggregate rates and daily series points.

#### Request body

None.

#### Response

`200 OK`.

```json
{
  "success": true,
  "data": {
    "object": "email.analytics",
    "windows": [
      {
        "days": 7,
        "rates": [
          { "name": "delivery_rate", "value": 0.985 },
          { "name": "open_rate", "value": 0.452 },
          { "name": "click_rate", "value": 0.127 },
          { "name": "bounce_rate", "value": 0.015 }
        ],
        "series": [
          {
            "date": "2026-08-29",
            "total": 100,
            "delivered": 98,
            "opened": 44,
            "clicked": 12,
            "bounced": 2
          }
        ]
      }
    ]
  }
}
```

---

## Email statuses

- `queued`
- `processing`
- `submitted`
- `delivered`
- `delayed`
- `bounced`
- `complained`
- `rejected`
- `failed`
- `canceled`

## Important scheduling behavior

Email scheduling is currently **one-time scheduling**. The API exposes a single `scheduled_at` timestamp for a message. There is no recurring-email schedule contract in this module.
