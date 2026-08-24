# Messaging

Customer dashboard HTTP contracts. Payloads and responses are generated from the public Go request and response types.

## Message Templates

### `POST /templates`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "name": "string",
  "html": "string",
  "alias": "string",
  "from": "string",
  "subject": "string",
  "reply_to": "string",
  "text": "string",
  "variables": [
    {
      "key": "string",
      "type": "string",
      "fallback_value": "string"
    }
  ]
}
```

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "object": "string",
    "id": "string"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /templates`

- Session: required.
- CSRF: not required.

#### Payload

Query parameters: `after`, `before`, `limit`.

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "object": "string",
    "data": [
      {
        "id": "string",
        "name": "string",
        "status": "string",
        "published_at": "2026-08-09T17:00:00Z",
        "created_at": "2026-08-09T17:00:00Z",
        "updated_at": "2026-08-09T17:00:00Z",
        "alias": "string"
      }
    ],
    "has_more": true
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /templates/:template`

- Session: required.
- CSRF: not required.

#### Payload

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "object": "string",
    "id": "string",
    "current_version_id": "string",
    "alias": "string",
    "name": "string",
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z",
    "status": "string",
    "published_at": "2026-08-09T17:00:00Z",
    "from": "string",
    "subject": "string",
    "reply_to": [
      "string"
    ],
    "html": "string",
    "text": "string",
    "variables": [
      {
        "id": "string",
        "key": "string",
        "type": "string",
        "fallback_value": "string",
        "created_at": "2026-08-09T17:00:00Z",
        "updated_at": "2026-08-09T17:00:00Z"
      }
    ],
    "has_unpublished_versions": true
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `PATCH /templates/:template`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "name": "string",
  "html": "string",
  "alias": "string",
  "from": "string",
  "subject": "string",
  "reply_to": "string",
  "text": "string",
  "variables": [
    {
      "key": "string",
      "type": "string",
      "fallback_value": "string"
    }
  ]
}
```

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "object": "string",
    "id": "string"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `DELETE /templates/:template`

- Session: required.
- CSRF: required for browser requests.

#### Payload

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "object": "string",
    "id": "string",
    "deleted": true
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /templates/:template/publish`

- Session: required.
- CSRF: required for browser requests.

#### Payload

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "object": "string",
    "id": "string"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /templates/:template/duplicate`

- Session: required.
- CSRF: required for browser requests.

#### Payload

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "object": "string",
    "id": "string"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /templates/:template/versions`

- Session: required.
- CSRF: not required.

#### Payload

Query parameters: `limit`, `offset`.

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": [
    {
      "id": "string",
      "team_id": "string",
      "template_id": "string",
      "version_number": 0,
      "from_email": "string",
      "from_name": "string",
      "reply_to_email": "string",
      "subject": "string",
      "html": "string",
      "text": "string",
      "variables": [
        {
          "key": "string",
          "type": "string",
          "fallback_value": "string"
        }
      ],
      "based_on_version_id": "string",
      "change_note": "string",
      "created_at": "2026-08-09T17:00:00Z"
    }
  ]
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /templates/:template/versions/:version_id`

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
    "template_id": "string",
    "version_number": 0,
    "from_email": "string",
    "from_name": "string",
    "reply_to_email": "string",
    "subject": "string",
    "html": "string",
    "text": "string",
    "variables": [
      {
        "key": "string",
        "type": "string",
        "fallback_value": "string"
      }
    ],
    "based_on_version_id": "string",
    "change_note": "string",
    "created_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /templates/:template/versions/:version_id/revert`

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
    "team_id": "string",
    "name": "string",
    "alias": "string",
    "current_version_id": "string",
    "published_version_id": "string",
    "published_at": "2026-08-09T17:00:00Z",
    "has_unpublished_changes": true,
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /templates/:template/preview`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "version_id": "string",
  "variables": {}
}
```

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "template_id": "string",
    "version_id": "string",
    "subject": "string",
    "html": "string",
    "text": "string",
    "from_email": "string",
    "from_name": "string",
    "reply_to": "string"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /templates/:template/test-send`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "to": "string",
  "version_id": "string",
  "variables": {}
}
```

#### Response — `202 Accepted`

```json
{
  "success": true,
  "data": "string"
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

## Sender Ids

### `GET /sender-ids`

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
      "name": "string",
      "country_code": "string",
      "purpose": "string",
      "status": "string",
      "provider": "string",
      "rejection_reason": "string",
      "approved_at": "2026-08-09T17:00:00Z",
      "rejected_at": "2026-08-09T17:00:00Z",
      "suspended_at": "2026-08-09T17:00:00Z",
      "created_by": "string",
      "created_at": "2026-08-09T17:00:00Z",
      "updated_at": "2026-08-09T17:00:00Z"
    }
  ]
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /sender-ids`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "name": "string",
  "country_code": "string",
  "purpose": "string",
  "provider": "string"
}
```

#### Response — `201 Created`

```json
{
  "success": true,
  "data": {
    "id": "string",
    "team_id": "string",
    "name": "string",
    "country_code": "string",
    "purpose": "string",
    "status": "string",
    "provider": "string",
    "rejection_reason": "string",
    "approved_at": "2026-08-09T17:00:00Z",
    "rejected_at": "2026-08-09T17:00:00Z",
    "suspended_at": "2026-08-09T17:00:00Z",
    "created_by": "string",
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /sender-ids/:sender_id`

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
    "name": "string",
    "country_code": "string",
    "purpose": "string",
    "status": "string",
    "provider": "string",
    "rejection_reason": "string",
    "approved_at": "2026-08-09T17:00:00Z",
    "rejected_at": "2026-08-09T17:00:00Z",
    "suspended_at": "2026-08-09T17:00:00Z",
    "created_by": "string",
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `DELETE /sender-ids/:sender_id`

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
    "team_id": "string",
    "name": "string",
    "country_code": "string",
    "purpose": "string",
    "status": "string",
    "provider": "string",
    "rejection_reason": "string",
    "approved_at": "2026-08-09T17:00:00Z",
    "rejected_at": "2026-08-09T17:00:00Z",
    "suspended_at": "2026-08-09T17:00:00Z",
    "created_by": "string",
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

## Domain

### `GET /domains`

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
      "name": "string",
      "provider": "string",
      "provider_account": "string",
      "region": "string",
      "provider_external_id": "string",
      "status": "string",
      "provider_status": "string",
      "records": [
        "string"
      ],
      "open_tracking": true,
      "click_tracking": true,
      "tracking_subdomain": "string",
      "active_tracking_subdomain": "string",
      "tls": "string",
      "capabilities": {
        "sending": true,
        "receiving": true
      },
      "custom_return_path": "string",
      "failure_reason": "string",
      "health_status": "string",
      "consecutive_health_failures": 0,
      "last_checked_at": "2026-08-09T17:00:00Z",
      "last_health_checked_at": "2026-08-09T17:00:00Z",
      "last_health_failure_at": "2026-08-09T17:00:00Z",
      "verified_at": "2026-08-09T17:00:00Z",
      "disabled_at": "2026-08-09T17:00:00Z",
      "created_by": "string",
      "created_at": "2026-08-09T17:00:00Z",
      "updated_at": "2026-08-09T17:00:00Z"
    }
  ]
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /domains`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "name": "string",
  "domain": "string",
  "region": "string",
  "custom_return_path": "string",
  "open_tracking": true,
  "click_tracking": true,
  "tracking_subdomain": "string",
  "tls": "string",
  "capabilities": {
    "sending": true,
    "receiving": true
  }
}
```

#### Response — `201 Created`

```json
{
  "success": true,
  "data": {}
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /domains/:domain_id`

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
    "name": "string",
    "provider": "string",
    "provider_account": "string",
    "region": "string",
    "provider_external_id": "string",
    "status": "string",
    "provider_status": "string",
    "records": [
      "string"
    ],
    "open_tracking": true,
    "click_tracking": true,
    "tracking_subdomain": "string",
    "active_tracking_subdomain": "string",
    "tls": "string",
    "capabilities": {
      "sending": true,
      "receiving": true
    },
    "custom_return_path": "string",
    "failure_reason": "string",
    "health_status": "string",
    "consecutive_health_failures": 0,
    "last_checked_at": "2026-08-09T17:00:00Z",
    "last_health_checked_at": "2026-08-09T17:00:00Z",
    "last_health_failure_at": "2026-08-09T17:00:00Z",
    "verified_at": "2026-08-09T17:00:00Z",
    "disabled_at": "2026-08-09T17:00:00Z",
    "created_by": "string",
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /domains/:domain_id/verify`

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
    "team_id": "string",
    "name": "string",
    "provider": "string",
    "provider_account": "string",
    "region": "string",
    "provider_external_id": "string",
    "status": "string",
    "provider_status": "string",
    "records": [
      "string"
    ],
    "open_tracking": true,
    "click_tracking": true,
    "tracking_subdomain": "string",
    "active_tracking_subdomain": "string",
    "tls": "string",
    "capabilities": {
      "sending": true,
      "receiving": true
    },
    "custom_return_path": "string",
    "failure_reason": "string",
    "health_status": "string",
    "consecutive_health_failures": 0,
    "last_checked_at": "2026-08-09T17:00:00Z",
    "last_health_checked_at": "2026-08-09T17:00:00Z",
    "last_health_failure_at": "2026-08-09T17:00:00Z",
    "verified_at": "2026-08-09T17:00:00Z",
    "disabled_at": "2026-08-09T17:00:00Z",
    "created_by": "string",
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `DELETE /domains/:domain_id`

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
    "team_id": "string",
    "name": "string",
    "provider": "string",
    "provider_account": "string",
    "region": "string",
    "provider_external_id": "string",
    "status": "string",
    "provider_status": "string",
    "records": [
      "string"
    ],
    "open_tracking": true,
    "click_tracking": true,
    "tracking_subdomain": "string",
    "active_tracking_subdomain": "string",
    "tls": "string",
    "capabilities": {
      "sending": true,
      "receiving": true
    },
    "custom_return_path": "string",
    "failure_reason": "string",
    "health_status": "string",
    "consecutive_health_failures": 0,
    "last_checked_at": "2026-08-09T17:00:00Z",
    "last_health_checked_at": "2026-08-09T17:00:00Z",
    "last_health_failure_at": "2026-08-09T17:00:00Z",
    "verified_at": "2026-08-09T17:00:00Z",
    "disabled_at": "2026-08-09T17:00:00Z",
    "created_by": "string",
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

## Sms

### `GET /sms/analytics`

- Session: required.
- CSRF: not required.

#### Payload

No JSON request body.

#### Response — `200 OK`

Returns 7-day, 30-day, and 90-day team-wide SMS delivery/failure rate windows with daily sparkline points, plus a 90-day delivery breakdown by destination country.

```json
{
  "success": true,
  "data": {
    "object": "sms.analytics",
    "windows": [
      {
        "days": 7,
        "rates": [{ "name": "delivery_rate", "value": 0.98 }],
        "series": [{ "date": "2026-08-24", "total": 10, "delivered": 9, "failed": 1 }]
      }
    ],
    "delivery_by_country": [
      { "country": "GH", "total": 10, "delivered": 9, "failed": 1 }
    ]
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /sms`

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
      "sender_id": "string",
      "to": "string",
      "from": "string",
      "body": "string",
      "status": "string",
      "provider_id": "string",
      "provider_message_id": "string",
      "segments": 0,
      "error_message": "string",
      "metadata": "string",
      "scheduled_at": "2026-08-09T17:00:00Z",
      "submitted_at": "2026-08-09T17:00:00Z",
      "delivered_at": "2026-08-09T17:00:00Z",
      "created_at": "2026-08-09T17:00:00Z",
      "updated_at": "2026-08-09T17:00:00Z",
      "destination_country": "string"
    }
  ]
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /sms`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "to": "string",
  "from": "string",
  "body": "string",
  "metadata": "string",
  "scheduled_at": "string"
}
```

#### Response — `202 Accepted`

```json
{
  "success": true,
  "data": {
    "id": "string",
    "team_id": "string",
    "sender_id": "string",
    "to": "string",
    "from": "string",
    "body": "string",
    "status": "string",
    "provider_id": "string",
    "provider_message_id": "string",
    "segments": 0,
    "error_message": "string",
    "metadata": "string",
    "scheduled_at": "2026-08-09T17:00:00Z",
    "submitted_at": "2026-08-09T17:00:00Z",
    "delivered_at": "2026-08-09T17:00:00Z",
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z",
    "destination_country": "string"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /sms/batch`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "messages": [
    {
      "to": "string",
      "from": "string",
      "body": "string",
      "metadata": "string",
      "scheduled_at": "string"
    }
  ]
}
```

#### Response — `202 Accepted`

```json
{
  "success": true,
  "data": [
    {
      "id": "string",
      "team_id": "string",
      "sender_id": "string",
      "to": "string",
      "from": "string",
      "body": "string",
      "status": "string",
      "provider_id": "string",
      "provider_message_id": "string",
      "segments": 0,
      "error_message": "string",
      "metadata": "string",
      "scheduled_at": "2026-08-09T17:00:00Z",
      "submitted_at": "2026-08-09T17:00:00Z",
      "delivered_at": "2026-08-09T17:00:00Z",
      "created_at": "2026-08-09T17:00:00Z",
      "updated_at": "2026-08-09T17:00:00Z",
      "destination_country": "string"
    }
  ]
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /sms/:message_id/cancel`

- Session: required.
- CSRF: required for browser requests.

#### Payload

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "object": "string",
    "id": "string"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `PATCH /sms/:message_id`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "scheduled_at": "string"
}
```

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "object": "string",
    "id": "string"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /sms/:message_id`

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
    "sender_id": "string",
    "to": "string",
    "from": "string",
    "body": "string",
    "status": "string",
    "provider_id": "string",
    "provider_message_id": "string",
    "segments": 0,
    "error_message": "string",
    "metadata": "string",
    "scheduled_at": "2026-08-09T17:00:00Z",
    "submitted_at": "2026-08-09T17:00:00Z",
    "delivered_at": "2026-08-09T17:00:00Z",
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z",
    "destination_country": "string"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /sms/:message_id/events`

- Session: required.
- CSRF: not required.

#### Payload

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "object": "string",
    "data": [
      {
        "id": "string",
        "type": "string",
        "occurred_at": "2026-08-09T17:00:00Z",
        "provider": "string",
        "code": "string",
        "message": "string"
      }
    ]
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /sms/:message_id/sync-status`

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
    "team_id": "string",
    "sender_id": "string",
    "to": "string",
    "from": "string",
    "body": "string",
    "status": "string",
    "provider_id": "string",
    "provider_message_id": "string",
    "segments": 0,
    "error_message": "string",
    "metadata": "string",
    "scheduled_at": "2026-08-09T17:00:00Z",
    "submitted_at": "2026-08-09T17:00:00Z",
    "delivered_at": "2026-08-09T17:00:00Z",
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z",
    "destination_country": "string"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

## Email

### `GET /emails/analytics`

- Session: required.
- CSRF: not required.

#### Payload

No JSON request body.

#### Response — `200 OK`

Returns 7-day, 30-day, and 90-day team-wide email delivery/open/click/bounce rate windows with daily sparkline points.

```json
{
  "success": true,
  "data": {
    "object": "email.analytics",
    "windows": [
      {
        "days": 7,
        "rates": [{ "name": "delivery_rate", "value": 0.98 }],
        "series": [{ "date": "2026-08-24", "total": 10, "delivered": 9, "opened": 5, "clicked": 2, "bounced": 1 }]
      }
    ]
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /emails`

- Session: required.
- CSRF: not required.

#### Payload

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {}
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /emails`

- Session: required.
- CSRF: required for browser requests.

#### Payload

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {}
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /emails/batch`

- Session: required.
- CSRF: required for browser requests.

#### Payload

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {}
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /emails/:message_id/cancel`

- Session: required.
- CSRF: required for browser requests.

#### Payload

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {}
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `PATCH /emails/:message_id`

- Session: required.
- CSRF: required for browser requests.

#### Payload

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {}
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /emails/:message_id`

- Session: required.
- CSRF: not required.

#### Payload

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {}
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /emails/:message_id/events`

- Session: required.
- CSRF: not required.

#### Payload

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {}
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.
