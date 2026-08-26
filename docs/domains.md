# Sender Domains API

Dashboard-facing HTTP contract for managing customer sender domains used by email delivery.

The routes and schemas in this document are derived from `server/internal/modules/domain` and the provider verification types used by that module. Internal persistence/provider fields that are not exposed by the HTTP handler are not documented as API fields.

## Conventions

- All routes are team-scoped through the authenticated tenant context.
- Access is controlled by sender-domain permissions; callers need read permission for list/get and create permission for create/verify. Delete requires delete permission.
- JSON responses use the standard `{ "success": true, "data": ... }` envelope.
- `POST /domains` normally returns `201 Created`. It can return `202 Accepted` while the customer's email infrastructure is still being provisioned.
- `POST /domains/:domain_id/verify` performs a provider/DNS verification check and returns the updated domain.
- `DELETE /domains/:domain_id` disables the sender domain and returns the updated domain representation.
- Domain IDs are UUID strings.
- Timestamps are RFC 3339/ISO-8601 strings.

---

## Sender domain object

The public `SenderDomain` response has this shape:

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "team_id": "650e8400-e29b-41d4-a716-446655440000",
  "name": "example.com",
  "region": "us-east-1",
  "provider_external_id": "provider-resource-id",
  "status": "pending",
  "provider_status": "verified",
  "records": [
    {
      "record": "DKIM",
      "name": "selector._domainkey.example.com",
      "value": "selector-value",
      "type": "TXT",
      "status": "pending",
      "ttl": "1800"
    },
    {
      "record": "SPF",
      "name": "send.example.com",
      "value": "feedback-smtp.us-east-1.amazonses.com",
      "type": "MX",
      "status": "pending",
      "ttl": "1800",
      "priority": 10
    }
  ],
  "tls": "opportunistic",
  "failure_reason": null,
  "health_status": "unknown",
  "consecutive_health_failures": 0,
  "last_checked_at": "2026-08-25T09:00:00Z",
  "last_health_checked_at": "2026-08-25T09:00:00Z",
  "last_health_failure_at": null,
  "verified_at": null,
  "disabled_at": null,
  "created_by": "750e8400-e29b-41d4-a716-446655440000",
  "created_at": "2026-08-25T08:59:58Z",
  "updated_at": "2026-08-25T09:00:00Z"
}
```

`provider` and `provider_account` are internal implementation details and are **not returned by the public domain API**.

### Sender domain fields

| Field | Type | Description |
| --- | --- | --- |
| `id` | string (UUID) | Sender domain ID. |
| `team_id` | string (UUID) | Owning team ID. |
| `name` | string | Normalized sender domain name. |
| `region` | string | Provider region. |
| `provider_external_id` | string, nullable | Provider-side resource identifier, when available. |
| `status` | string | Verification/provisioning status. |
| `provider_status` | string, nullable | Provider-reported status, when available. |
| `records` | array | DNS verification records required for the domain. |
| `tls` | string | TLS mode. `opportunistic` or `enforced`. |
| `failure_reason` | string, nullable | Reason the latest verification/health operation failed, when available. |
| `health_status` | string | Domain health: `unknown`, `healthy`, or `degraded`. |
| `consecutive_health_failures` | integer | Consecutive automated health-check failures. |
| `last_checked_at` | string, nullable | Last sender-domain verification/reconciliation check. |
| `last_health_checked_at` | string, nullable | Last automated health check. |
| `last_health_failure_at` | string, nullable | Most recent health-check failure. |
| `verified_at` | string, nullable | Time the domain became verified. |
| `disabled_at` | string, nullable | Time the domain was disabled. |
| `created_by` | string, nullable | User ID that created the domain. |
| `created_at` | string | Creation time. |
| `updated_at` | string | Last update time. |

`custom_return_path` is intentionally not exposed by the public JSON response.

### Verification record fields

```json
{
  "record": "DKIM",
  "name": "selector._domainkey.example.com",
  "value": "selector-value",
  "type": "TXT",
  "status": "pending",
  "ttl": "1800",
  "priority": 10
}
```

| Field | Type | Description |
| --- | --- | --- |
| `record` | string | Verification purpose. Currently `DKIM` or `SPF`. |
| `name` | string | DNS record name. |
| `value` | string | DNS record value/content. |
| `type` | string | DNS record type. Currently `TXT` or `MX`. |
| `status` | string | Record status: `pending`, `verified`, or `failed`. |
| `ttl` | string | DNS TTL supplied by the provider. |
| `priority` | integer, nullable | MX priority when applicable. Omitted when not applicable. |

---

## List sender domains

### `GET /domains`

Returns all sender domains belonging to the current team.

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
      "team_id": "650e8400-e29b-41d4-a716-446655440000",
      "name": "example.com",
      "region": "us-east-1",
      "status": "pending",
      "records": [],
      "tls": "opportunistic",
      "health_status": "unknown",
      "consecutive_health_failures": 0,
      "created_at": "2026-08-25T08:59:58Z",
      "updated_at": "2026-08-25T08:59:58Z"
    }
  ]
}
```

---

## Create a sender domain

### `POST /domains`

Creates a sender domain and starts provider provisioning and DNS verification.

#### Request body

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `name` | string | Yes | Domain name. The server normalizes it to lowercase and removes an optional `http://`, `https://`, trailing dot, and path before validation. |
| `region` | string | Yes | SES region. Currently `us-east-1` or `eu-north-1`. |
| `tls` | string | No | TLS mode: `opportunistic` or `enforced`. Defaults to `opportunistic`. |

```json
{
  "name": "example.com",
  "region": "us-east-1",
  "tls": "opportunistic"
}
```

The normalized domain must be a valid DNS domain name and no longer than 253 characters. A duplicate sender domain for the team returns `409 Conflict`.

#### Response: provisioning complete

`201 Created`.

```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "team_id": "650e8400-e29b-41d4-a716-446655440000",
    "name": "example.com",
    "region": "us-east-1",
    "status": "pending",
    "records": [],
    "tls": "opportunistic",
    "health_status": "unknown",
    "consecutive_health_failures": 0,
    "created_at": "2026-08-25T08:59:58Z",
    "updated_at": "2026-08-25T09:00:00Z"
  }
}
```

#### Response: email infrastructure still provisioning

`202 Accepted` with `Retry-After: 10`.

```json
{
  "success": true,
  "data": {
    "status": "provisioning",
    "message": "Customer email infrastructure is being prepared",
    "retry_after_seconds": 10
  }
}
```

When this response is returned, the domain has not necessarily been created yet. The client should wait and retry its domain state lookup rather than assuming verification records are available immediately.

---

## Get a sender domain

### `GET /domains/:domain_id`

Returns one sender domain belonging to the current team.

#### Path parameters

| Parameter | Type | Description |
| --- | --- | --- |
| `domain_id` | string (UUID) | Sender domain ID. |

#### Request body

None.

#### Response

`200 OK` with the [Sender domain object](#sender-domain-object).

```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "team_id": "650e8400-e29b-41d4-a716-446655440000",
    "name": "example.com",
    "region": "us-east-1",
    "status": "pending",
    "records": [],
    "tls": "opportunistic",
    "health_status": "unknown",
    "consecutive_health_failures": 0,
    "created_at": "2026-08-25T08:59:58Z",
    "updated_at": "2026-08-25T08:59:58Z"
  }
}
```

An invalid UUID returns `400 Bad Request`; a valid UUID that does not belong to the current team returns `404 Not Found`.

---

## Verify a sender domain

### `POST /domains/:domain_id/verify`

Runs provider and DNS verification for the domain and persists the resulting status and verification records. For an already verified domain, this operation also performs a health observation.

#### Path parameters

| Parameter | Type | Description |
| --- | --- | --- |
| `domain_id` | string (UUID) | Sender domain ID. |

#### Request body

None.

#### Response

`200 OK` with the updated [Sender domain object](#sender-domain-object).

```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "team_id": "650e8400-e29b-41d4-a716-446655440000",
    "name": "example.com",
    "region": "us-east-1",
    "status": "verified",
    "provider_status": "verified",
    "records": [
      {
        "record": "DKIM",
        "name": "selector._domainkey.example.com",
        "value": "selector-value",
        "type": "TXT",
        "status": "verified",
        "ttl": "1800"
      }
    ],
    "tls": "opportunistic",
    "health_status": "healthy",
    "consecutive_health_failures": 0,
    "verified_at": "2026-08-25T09:05:00Z",
    "created_at": "2026-08-25T08:59:58Z",
    "updated_at": "2026-08-25T09:05:00Z"
  }
}
```

A disabled domain cannot be verified and returns `409 Conflict`. A verification failure is persisted on the domain and returned as the updated domain when the verification operation itself completes with a domain-level failure state.

---

## Disable a sender domain

### `DELETE /domains/:domain_id`

Disables the sender domain for the team and removes its active provider association where supported. This is a logical disable operation from the public API perspective.

#### Path parameters

| Parameter | Type | Description |
| --- | --- | --- |
| `domain_id` | string (UUID) | Sender domain ID. |

#### Request body

None.

#### Response

`200 OK` with the disabled [Sender domain object](#sender-domain-object).

```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "team_id": "650e8400-e29b-41d4-a716-446655440000",
    "name": "example.com",
    "region": "us-east-1",
    "status": "disabled",
    "records": [],
    "tls": "opportunistic",
    "health_status": "unknown",
    "disabled_at": "2026-08-25T10:00:00Z",
    "created_at": "2026-08-25T08:59:58Z",
    "updated_at": "2026-08-25T10:00:00Z"
  }
}
```

A valid domain ID that is not owned by the current team returns `404 Not Found`.

---

## Status values

### Verification/provisioning status

The implementation defines:

- `not_started`
- `pending`
- `verified`
- `partially_verified`
- `partially_failed`
- `failed`
- `temporary_failure`
- `disabled`

The normal customer flow is `pending` → `verified`, with `failed` or temporary/pending states possible while provider/DNS checks are unresolved.

### Health status

- `unknown` — no successful health observation is currently available.
- `healthy` — automated health checks are passing.
- `degraded` — repeated health checks have failed or the latest health observation is unhealthy.

Automated reconciliation continues checking pending domains and verified-domain health in the background. The dashboard should use the returned timestamps/statuses rather than assuming verification is instantaneous.

## Supported provider regions

The current provider deployment accepts:

- `us-east-1`
- `eu-north-1`

The value is normalized to lowercase before validation.

## TLS modes

- `opportunistic` — default.
- `enforced`.

Although the internal domain service contains an update-configuration method, **there is currently no public `PATCH /domains/:domain_id` route registered**. Frontend code should not build against a domain-update endpoint until one is exposed by the route registration.
