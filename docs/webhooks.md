# Webhooks

Customer dashboard HTTP contracts. Payloads and responses are generated from the public Go request and response types.

## Available Events

The following events can be subscribed to via webhooks:

### SMS Events
- `sms.submitted` — SMS message submitted for delivery
- `sms.sent` — SMS message sent
- `sms.delivered` — SMS message delivered successfully
- `sms.undelivered` — SMS message could not be delivered
- `sms.failed` — SMS message failed

### Email Events
- `email.submitted` — Email submitted for delivery
- `email.delivered` — Email delivered successfully
- `email.delayed` — Email delivery delayed
- `email.bounced` — Email bounced
- `email.complained` — Recipient marked email as spam/complaint
- `email.rejected` — Email rejected by provider
- `email.failed` — Email delivery failed
- `email.opened` — Email opened by recipient
- `email.clicked` — Link in email clicked by recipient
- `email.subscription_changed` — Email subscription status changed

### Contact Events
- `contact.created` — New contact created
- `contact.updated` — Contact updated
- `contact.deleted` — Contact deleted

### Suppression Events
- `suppression.created` — New suppression added
- `suppression.deleted` — Suppression removed

### Broadcast Events
- `broadcast.scheduled` — Broadcast scheduled
- `broadcast.queued` — Broadcast queued for sending
- `broadcast.sent` — Broadcast sent
- `broadcast.failed` — Broadcast failed
- `broadcast.canceled` — Broadcast canceled

### Test Event
- `webhook.test` — Test event (used for testing webhook endpoints)

## Webhooks

### `POST /webhook-endpoints`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "url": "string",
  "subscribed_events": [
    "string"
  ]
}
```

#### Response — `201 Created`

```json
{
  "success": true,
  "data": {
    "signing_secret": "string"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /webhook-endpoints`

- Session: required.
- CSRF: not required.

#### Payload

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": [
    {
      "id": "string",
      "team_id": "string",
      "url": "string",
      "enabled": true,
      "subscribed_events": [
        "string"
      ],
      "created_at": "2026-08-09T17:00:00Z",
      "updated_at": "2026-08-09T17:00:00Z",
      "disabled_at": "2026-08-09T17:00:00Z",
      "consecutive_failures": 0,
      "last_failure_at": "2026-08-09T17:00:00Z",
      "disabled_reason": "string"
    }
  ]
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /webhook-endpoints/:endpoint_id`

- Session: required.
- CSRF: not required.

#### Payload

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "id": "string",
    "team_id": "string",
    "url": "string",
    "enabled": true,
    "subscribed_events": [
      "string"
    ],
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z",
    "disabled_at": "2026-08-09T17:00:00Z",
    "consecutive_failures": 0,
    "last_failure_at": "2026-08-09T17:00:00Z",
    "disabled_reason": "string"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `PATCH /webhook-endpoints/:endpoint_id`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "url": "string",
  "enabled": true,
  "subscribed_events": [
    "string"
  ]
}
```

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "id": "string",
    "team_id": "string",
    "url": "string",
    "enabled": true,
    "subscribed_events": [
      "string"
    ],
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z",
    "disabled_at": "2026-08-09T17:00:00Z",
    "consecutive_failures": 0,
    "last_failure_at": "2026-08-09T17:00:00Z",
    "disabled_reason": "string"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `DELETE /webhook-endpoints/:endpoint_id`

- Session: required.
- CSRF: required for browser requests.

#### Payload

No JSON request body.

#### Response — `204 No Content`

No response body.

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /webhook-endpoints/:endpoint_id/test`

- Session: required.
- CSRF: required for browser requests.

#### Payload

No JSON request body.

#### Response — `201 Created`

```json
{
  "success": true,
  "data": {
    "id": "string",
    "event_id": "string",
    "endpoint_id": "string",
    "status": "string",
    "attempt_count": 0,
    "next_attempt_at": "2026-08-09T17:00:00Z",
    "last_attempt_at": "2026-08-09T17:00:00Z",
    "response_status": 0,
    "response_body": "string",
    "last_error": "string",
    "delivered_at": "2026-08-09T17:00:00Z",
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /webhook-endpoints/:endpoint_id/rotate-secret`

- Session: required.
- CSRF: required for browser requests.

#### Payload

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "signing_secret": "string"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /webhook-events`

- Session: required.
- CSRF: not required.

#### Payload

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": [
    {
      "id": "string",
      "team_id": "string",
      "type": "string",
      "object_type": "string",
      "object_id": "string",
      "payload": "string",
      "occurred_at": "2026-08-09T17:00:00Z",
      "created_at": "2026-08-09T17:00:00Z"
    }
  ]
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /webhook-events/:event_id`

- Session: required.
- CSRF: not required.

#### Payload

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "id": "string",
    "team_id": "string",
    "type": "string",
    "object_type": "string",
    "object_id": "string",
    "payload": "string",
    "occurred_at": "2026-08-09T17:00:00Z",
    "created_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /webhook-deliveries/:delivery_id`

- Session: required.
- CSRF: not required.

#### Payload

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "id": "string",
    "event_id": "string",
    "endpoint_id": "string",
    "status": "string",
    "attempt_count": 0,
    "next_attempt_at": "2026-08-09T17:00:00Z",
    "last_attempt_at": "2026-08-09T17:00:00Z",
    "response_status": 0,
    "response_body": "string",
    "last_error": "string",
    "delivered_at": "2026-08-09T17:00:00Z",
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /webhook-deliveries/:delivery_id/retry`

- Session: required.
- CSRF: required for browser requests.

#### Payload

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "id": "string",
    "event_id": "string",
    "endpoint_id": "string",
    "status": "string",
    "attempt_count": 0,
    "next_attempt_at": "2026-08-09T17:00:00Z",
    "last_attempt_at": "2026-08-09T17:00:00Z",
    "response_status": 0,
    "response_body": "string",
    "last_error": "string",
    "delivered_at": "2026-08-09T17:00:00Z",
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.
