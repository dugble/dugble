# Campaigns

Customer dashboard HTTP contracts. Payloads and responses are generated from the public Go request and response types.

## Campaign

### `POST /campaigns`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "name": "string",
  "segment_id": "string",
  "sender_id": "string",
  "body": "string",
  "rate_limit_per_second": 0,
  "daily_send_limit": 0
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
    "status": "string",
    "segment_id": "string",
    "sender_id": "string",
    "body": "string",
    "scheduled_at": "2026-08-09T17:00:00Z",
    "queued_at": "2026-08-09T17:00:00Z",
    "canceled_at": "2026-08-09T17:00:00Z",
    "materialized_at": "2026-08-09T17:00:00Z",
    "sent_at": "2026-08-09T17:00:00Z",
    "audience_count": 0,
    "eligible_count": 0,
    "excluded_count": 0,
    "failed_count": 0,
    "estimated_segments": 0,
    "estimated_cost_units": 0,
    "estimated_billable_cost_units": 0,
    "preflight_allowance_segments": 0,
    "actual_segments": 0,
    "actual_charge_units": 0,
    "currency": "string",
    "preflight_balance_units": 0,
    "preflight_at": "2026-08-09T17:00:00Z",
    "rate_limit_per_second": 0,
    "daily_send_limit": 0,
    "revision": 0,
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /campaigns`

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
      "status": "string",
      "segment_id": "string",
      "sender_id": "string",
      "body": "string",
      "scheduled_at": "2026-08-09T17:00:00Z",
      "queued_at": "2026-08-09T17:00:00Z",
      "canceled_at": "2026-08-09T17:00:00Z",
      "materialized_at": "2026-08-09T17:00:00Z",
      "sent_at": "2026-08-09T17:00:00Z",
      "audience_count": 0,
      "eligible_count": 0,
      "excluded_count": 0,
      "failed_count": 0,
      "estimated_segments": 0,
      "estimated_cost_units": 0,
      "estimated_billable_cost_units": 0,
      "preflight_allowance_segments": 0,
      "actual_segments": 0,
      "actual_charge_units": 0,
      "currency": "string",
      "preflight_balance_units": 0,
      "preflight_at": "2026-08-09T17:00:00Z",
      "rate_limit_per_second": 0,
      "daily_send_limit": 0,
      "revision": 0,
      "created_at": "2026-08-09T17:00:00Z",
      "updated_at": "2026-08-09T17:00:00Z"
    }
  ]
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /sms/opt-outs`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "phone": "string",
  "source": "string",
  "source_id": "string"
}
```

#### Response — `201 Created`

```json
{
  "success": true,
  "data": {
    "id": "string",
    "contact_id": "string",
    "phone": "string",
    "action": "string",
    "source": "string",
    "source_id": "string",
    "recorded_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /campaigns/:campaign`

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
    "status": "string",
    "segment_id": "string",
    "sender_id": "string",
    "body": "string",
    "scheduled_at": "2026-08-09T17:00:00Z",
    "queued_at": "2026-08-09T17:00:00Z",
    "canceled_at": "2026-08-09T17:00:00Z",
    "materialized_at": "2026-08-09T17:00:00Z",
    "sent_at": "2026-08-09T17:00:00Z",
    "audience_count": 0,
    "eligible_count": 0,
    "excluded_count": 0,
    "failed_count": 0,
    "estimated_segments": 0,
    "estimated_cost_units": 0,
    "estimated_billable_cost_units": 0,
    "preflight_allowance_segments": 0,
    "actual_segments": 0,
    "actual_charge_units": 0,
    "currency": "string",
    "preflight_balance_units": 0,
    "preflight_at": "2026-08-09T17:00:00Z",
    "rate_limit_per_second": 0,
    "daily_send_limit": 0,
    "revision": 0,
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `PATCH /campaigns/:campaign`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "revision": 0,
  "name": "string",
  "segment_id": "string",
  "sender_id": "string",
  "body": "string",
  "rate_limit_per_second": 0,
  "daily_send_limit": 0
}
```

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "id": "string",
    "team_id": "string",
    "name": "string",
    "status": "string",
    "segment_id": "string",
    "sender_id": "string",
    "body": "string",
    "scheduled_at": "2026-08-09T17:00:00Z",
    "queued_at": "2026-08-09T17:00:00Z",
    "canceled_at": "2026-08-09T17:00:00Z",
    "materialized_at": "2026-08-09T17:00:00Z",
    "sent_at": "2026-08-09T17:00:00Z",
    "audience_count": 0,
    "eligible_count": 0,
    "excluded_count": 0,
    "failed_count": 0,
    "estimated_segments": 0,
    "estimated_cost_units": 0,
    "estimated_billable_cost_units": 0,
    "preflight_allowance_segments": 0,
    "actual_segments": 0,
    "actual_charge_units": 0,
    "currency": "string",
    "preflight_balance_units": 0,
    "preflight_at": "2026-08-09T17:00:00Z",
    "rate_limit_per_second": 0,
    "daily_send_limit": 0,
    "revision": 0,
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `DELETE /campaigns/:campaign`

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
    "status": "string",
    "segment_id": "string",
    "sender_id": "string",
    "body": "string",
    "scheduled_at": "2026-08-09T17:00:00Z",
    "queued_at": "2026-08-09T17:00:00Z",
    "canceled_at": "2026-08-09T17:00:00Z",
    "materialized_at": "2026-08-09T17:00:00Z",
    "sent_at": "2026-08-09T17:00:00Z",
    "audience_count": 0,
    "eligible_count": 0,
    "excluded_count": 0,
    "failed_count": 0,
    "estimated_segments": 0,
    "estimated_cost_units": 0,
    "estimated_billable_cost_units": 0,
    "preflight_allowance_segments": 0,
    "actual_segments": 0,
    "actual_charge_units": 0,
    "currency": "string",
    "preflight_balance_units": 0,
    "preflight_at": "2026-08-09T17:00:00Z",
    "rate_limit_per_second": 0,
    "daily_send_limit": 0,
    "revision": 0,
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /campaigns/:campaign/duplicate`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "name": "string"
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
    "status": "string",
    "segment_id": "string",
    "sender_id": "string",
    "body": "string",
    "scheduled_at": "2026-08-09T17:00:00Z",
    "queued_at": "2026-08-09T17:00:00Z",
    "canceled_at": "2026-08-09T17:00:00Z",
    "materialized_at": "2026-08-09T17:00:00Z",
    "sent_at": "2026-08-09T17:00:00Z",
    "audience_count": 0,
    "eligible_count": 0,
    "excluded_count": 0,
    "failed_count": 0,
    "estimated_segments": 0,
    "estimated_cost_units": 0,
    "estimated_billable_cost_units": 0,
    "preflight_allowance_segments": 0,
    "actual_segments": 0,
    "actual_charge_units": 0,
    "currency": "string",
    "preflight_balance_units": 0,
    "preflight_at": "2026-08-09T17:00:00Z",
    "rate_limit_per_second": 0,
    "daily_send_limit": 0,
    "revision": 0,
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /campaigns/:campaign/preview`

- Session: required.
- CSRF: required for browser requests.

#### Payload

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": "string"
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /campaigns/:campaign/send`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "scheduled_at": "2026-08-09T17:00:00Z"
}
```

#### Response — `202 Accepted`

```json
{
  "success": true,
  "data": {
    "id": "string",
    "team_id": "string",
    "name": "string",
    "status": "string",
    "segment_id": "string",
    "sender_id": "string",
    "body": "string",
    "scheduled_at": "2026-08-09T17:00:00Z",
    "queued_at": "2026-08-09T17:00:00Z",
    "canceled_at": "2026-08-09T17:00:00Z",
    "materialized_at": "2026-08-09T17:00:00Z",
    "sent_at": "2026-08-09T17:00:00Z",
    "audience_count": 0,
    "eligible_count": 0,
    "excluded_count": 0,
    "failed_count": 0,
    "estimated_segments": 0,
    "estimated_cost_units": 0,
    "estimated_billable_cost_units": 0,
    "preflight_allowance_segments": 0,
    "actual_segments": 0,
    "actual_charge_units": 0,
    "currency": "string",
    "preflight_balance_units": 0,
    "preflight_at": "2026-08-09T17:00:00Z",
    "rate_limit_per_second": 0,
    "daily_send_limit": 0,
    "revision": 0,
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /campaigns/:campaign/schedule`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "scheduled_at": "2026-08-09T17:00:00Z"
}
```

#### Response — `202 Accepted`

```json
{
  "success": true,
  "data": {
    "id": "string",
    "team_id": "string",
    "name": "string",
    "status": "string",
    "segment_id": "string",
    "sender_id": "string",
    "body": "string",
    "scheduled_at": "2026-08-09T17:00:00Z",
    "queued_at": "2026-08-09T17:00:00Z",
    "canceled_at": "2026-08-09T17:00:00Z",
    "materialized_at": "2026-08-09T17:00:00Z",
    "sent_at": "2026-08-09T17:00:00Z",
    "audience_count": 0,
    "eligible_count": 0,
    "excluded_count": 0,
    "failed_count": 0,
    "estimated_segments": 0,
    "estimated_cost_units": 0,
    "estimated_billable_cost_units": 0,
    "preflight_allowance_segments": 0,
    "actual_segments": 0,
    "actual_charge_units": 0,
    "currency": "string",
    "preflight_balance_units": 0,
    "preflight_at": "2026-08-09T17:00:00Z",
    "rate_limit_per_second": 0,
    "daily_send_limit": 0,
    "revision": 0,
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /campaigns/:campaign/cancel`

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
    "status": "string",
    "segment_id": "string",
    "sender_id": "string",
    "body": "string",
    "scheduled_at": "2026-08-09T17:00:00Z",
    "queued_at": "2026-08-09T17:00:00Z",
    "canceled_at": "2026-08-09T17:00:00Z",
    "materialized_at": "2026-08-09T17:00:00Z",
    "sent_at": "2026-08-09T17:00:00Z",
    "audience_count": 0,
    "eligible_count": 0,
    "excluded_count": 0,
    "failed_count": 0,
    "estimated_segments": 0,
    "estimated_cost_units": 0,
    "estimated_billable_cost_units": 0,
    "preflight_allowance_segments": 0,
    "actual_segments": 0,
    "actual_charge_units": 0,
    "currency": "string",
    "preflight_balance_units": 0,
    "preflight_at": "2026-08-09T17:00:00Z",
    "rate_limit_per_second": 0,
    "daily_send_limit": 0,
    "revision": 0,
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /campaigns/:campaign/recipients`

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
      "campaign_id": "string",
      "contact_id": "string",
      "phone": "string",
      "phone_country": "string",
      "contact_snapshot": {},
      "status": "string",
      "exclusion_reason": "string",
      "sms_message_id": "string",
      "created_at": "2026-08-09T17:00:00Z",
      "queued_at": "2026-08-09T17:00:00Z",
      "rendered_body": "string",
      "attempt_count": 0,
      "failure_code": "string",
      "failure_message": "string",
      "encoding": "string",
      "estimated_segments": 0,
      "estimated_unit_cost_units": 0,
      "estimated_cost_units": 0,
      "actual_segments": 0,
      "actual_charge_units": 0
    }
  ]
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /campaigns/:campaign/cost-estimate`

- Session: required.
- CSRF: not required.

#### Payload

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "campaign_id": "string",
    "currency": "string",
    "recipients": 0,
    "estimated_segments": 0,
    "estimated_cost_units": 0,
    "estimated_billable_cost_units": 0,
    "preflight_allowance_segments": 0,
    "minimum_recipient_cost_units": 0,
    "maximum_recipient_cost_units": 0,
    "actual_segments": 0,
    "actual_charge_units": 0,
    "preflight_balance_units": 0,
    "preflight_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /campaigns/:campaign/exclusions`

- Session: required.
- CSRF: not required.

#### Payload

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "campaign_id": "string",
    "total": 0,
    "reasons": 0
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /campaigns/:campaign/analytics`

- Session: required.
- CSRF: not required.

#### Payload

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "campaign_id": "string",
    "audience": 0,
    "eligible": 0,
    "excluded": 0,
    "queued": 0,
    "failed": 0,
    "sent": 0,
    "delivered": 0,
    "delivery_failed": 0,
    "estimated_segments": 0,
    "estimated_cost_units": 0,
    "estimated_billable_cost_units": 0,
    "actual_segments": 0,
    "actual_charge_units": 0,
    "currency": "string"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

## Broadcast

### `POST /broadcasts`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "name": "string",
  "segment_id": "string",
  "topic_id": "string",
  "template": "string",
  "variable_bindings": {}
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
    "status": "string",
    "segment_id": "string",
    "topic_id": "string",
    "template_id": "string",
    "template_version_id": "string",
    "variable_bindings": {},
    "scheduled_at": "2026-08-09T17:00:00Z",
    "queued_at": "2026-08-09T17:00:00Z",
    "sent_at": "2026-08-09T17:00:00Z",
    "canceled_at": "2026-08-09T17:00:00Z",
    "audience_count": 0,
    "eligible_count": 0,
    "suppressed_count": 0,
    "queued_count": 0,
    "failed_count": 0,
    "revision": 0,
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /broadcasts`

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
      "status": "string",
      "segment_id": "string",
      "topic_id": "string",
      "template_id": "string",
      "template_version_id": "string",
      "variable_bindings": {},
      "scheduled_at": "2026-08-09T17:00:00Z",
      "queued_at": "2026-08-09T17:00:00Z",
      "sent_at": "2026-08-09T17:00:00Z",
      "canceled_at": "2026-08-09T17:00:00Z",
      "audience_count": 0,
      "eligible_count": 0,
      "suppressed_count": 0,
      "queued_count": 0,
      "failed_count": 0,
      "revision": 0,
      "created_at": "2026-08-09T17:00:00Z",
      "updated_at": "2026-08-09T17:00:00Z"
    }
  ]
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /broadcasts/:broadcast`

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
    "status": "string",
    "segment_id": "string",
    "topic_id": "string",
    "template_id": "string",
    "template_version_id": "string",
    "variable_bindings": {},
    "scheduled_at": "2026-08-09T17:00:00Z",
    "queued_at": "2026-08-09T17:00:00Z",
    "sent_at": "2026-08-09T17:00:00Z",
    "canceled_at": "2026-08-09T17:00:00Z",
    "audience_count": 0,
    "eligible_count": 0,
    "suppressed_count": 0,
    "queued_count": 0,
    "failed_count": 0,
    "revision": 0,
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `PATCH /broadcasts/:broadcast`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "revision": 0,
  "name": "string",
  "segment_id": "string",
  "topic_id": "string",
  "template": "string",
  "variable_bindings": {}
}
```

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "id": "string",
    "team_id": "string",
    "name": "string",
    "status": "string",
    "segment_id": "string",
    "topic_id": "string",
    "template_id": "string",
    "template_version_id": "string",
    "variable_bindings": {},
    "scheduled_at": "2026-08-09T17:00:00Z",
    "queued_at": "2026-08-09T17:00:00Z",
    "sent_at": "2026-08-09T17:00:00Z",
    "canceled_at": "2026-08-09T17:00:00Z",
    "audience_count": 0,
    "eligible_count": 0,
    "suppressed_count": 0,
    "queued_count": 0,
    "failed_count": 0,
    "revision": 0,
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `DELETE /broadcasts/:broadcast`

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
    "status": "string",
    "segment_id": "string",
    "topic_id": "string",
    "template_id": "string",
    "template_version_id": "string",
    "variable_bindings": {},
    "scheduled_at": "2026-08-09T17:00:00Z",
    "queued_at": "2026-08-09T17:00:00Z",
    "sent_at": "2026-08-09T17:00:00Z",
    "canceled_at": "2026-08-09T17:00:00Z",
    "audience_count": 0,
    "eligible_count": 0,
    "suppressed_count": 0,
    "queued_count": 0,
    "failed_count": 0,
    "revision": 0,
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /broadcasts/:broadcast/send`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "scheduled_at": "2026-08-09T17:00:00Z"
}
```

#### Response — `202 Accepted`

```json
{
  "success": true,
  "data": {
    "id": "string",
    "team_id": "string",
    "name": "string",
    "status": "string",
    "segment_id": "string",
    "topic_id": "string",
    "template_id": "string",
    "template_version_id": "string",
    "variable_bindings": {},
    "scheduled_at": "2026-08-09T17:00:00Z",
    "queued_at": "2026-08-09T17:00:00Z",
    "sent_at": "2026-08-09T17:00:00Z",
    "canceled_at": "2026-08-09T17:00:00Z",
    "audience_count": 0,
    "eligible_count": 0,
    "suppressed_count": 0,
    "queued_count": 0,
    "failed_count": 0,
    "revision": 0,
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /broadcasts/:broadcast/cancel`

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
    "status": "string",
    "segment_id": "string",
    "topic_id": "string",
    "template_id": "string",
    "template_version_id": "string",
    "variable_bindings": {},
    "scheduled_at": "2026-08-09T17:00:00Z",
    "queued_at": "2026-08-09T17:00:00Z",
    "sent_at": "2026-08-09T17:00:00Z",
    "canceled_at": "2026-08-09T17:00:00Z",
    "audience_count": 0,
    "eligible_count": 0,
    "suppressed_count": 0,
    "queued_count": 0,
    "failed_count": 0,
    "revision": 0,
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /broadcasts/:broadcast/duplicate`

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

### `POST /broadcasts/:broadcast/preview`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
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

### `GET /broadcasts/:broadcast/recipients`

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
      "broadcast_id": "string",
      "contact_id": "string",
      "email": "string",
      "first_name": "string",
      "last_name": "string",
      "contact_snapshot": {},
      "status": "string",
      "exclusion_reason": "string",
      "email_message_id": "string",
      "created_at": "2026-08-09T17:00:00Z",
      "queued_at": "2026-08-09T17:00:00Z"
    }
  ]
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /broadcasts/:broadcast/exclusions`

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
    "broadcast_id": "string",
    "total": 0,
    "reasons": 0
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /broadcasts/:broadcast/analytics`

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
    "broadcast_id": "string",
    "audience": 0,
    "eligible": 0,
    "excluded": 0,
    "queued": 0,
    "delivered": 0,
    "bounced": 0,
    "complained": 0,
    "failed": 0,
    "opened": 0,
    "clicked": 0
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.
