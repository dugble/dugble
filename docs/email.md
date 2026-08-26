# Email API

Dashboard-facing HTTP contract for the Email API.

## Conventions

- Mutating requests (`POST`, `PATCH`) require an `Idempotency-Key` header.
- Successful responses use the repository's standard JSON envelope.
- Collection endpoints use `limit` and `offset` for pagination.
- Email addresses may be supplied as an object or a string.

## List emails

### `GET /emails`

Returns email message summaries.

| Parameter | Type | Description |
| --- | --- | --- |
| `limit` | integer | Maximum number of results. |
| `offset` | integer | Number of results to skip. |

## Send an email

### `POST /emails`

Creates and queues a single email for delivery.

**Required header:** `Idempotency-Key`

```json
{
  "from": {"email": "sender@example.com", "name": "Sender"},
  "to": ["Recipient <recipient@example.com>"],
  "subject": "Welcome",
  "html": "<p>Hello</p>",
  "text": "Hello",
  "scheduled_at": "2026-08-25T09:00:00Z",
  "metadata": {"contact_id": "string"}
}
```

Returns `202 Accepted` and a `Location` header pointing to `/emails/:id`.

## Send emails in bulk

### `POST /emails/batch`

Queues multiple emails for delivery. The request accepts either a top-level array or an object containing `messages`.

**Required header:** `Idempotency-Key`

```json
{"messages": [{"to": "recipient@example.com", "subject": "Hello", "html": "<p>Hello</p>"}]}
```

Returns `202 Accepted`.

## Get an email

### `GET /emails/:message_id`

Returns the complete representation of an email message, including recipients, content, provider message ID, status, timestamps, scheduled time, and tags.

## Update an email

### `PATCH /emails/:message_id`

Updates an eligible email. The documented update is scheduling.

**Required header:** `Idempotency-Key`

```json
{"scheduled_at": "2026-08-25T09:00:00Z"}
```

## Cancel an email

### `POST /emails/:message_id/cancel`

Cancels an eligible email. No request body.

**Required header:** `Idempotency-Key`

## List email events

### `GET /emails/:message_id/events`

Returns delivery events recorded for an email.

| Parameter | Type | Description |
| --- | --- | --- |
| `limit` | integer | Maximum number of events to return. |

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
