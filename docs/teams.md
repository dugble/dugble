# Teams

Customer dashboard HTTP contracts. Payloads and responses are generated from the public Go request and response types.

## Enums

The team APIs use the following string enum values:

| Field | Values |
| --- | --- |
| `team.status` | `active`, `disabled` |
| `team.user_role` | `owner`, `admin`, `member` |
| `member.role` | `owner`, `admin`, `member` |
| `member.status` | `active`, `suspended`, `invited` |
| `invitation.status` | `pending`, `accepted`, `declined`, `revoked` |
| `invitation.role` | `owner`, `admin`, `member` |

## Team

### `GET /teams`

- Session: required.
- CSRF: not required.

#### Query parameters

| Parameter | Type | Default | Description |
| --- | --- | --- | --- |
| `page` | positive integer | `1` | Page number to return. |
| `limit` | integer from `1` to `100` | `20` | Maximum number of teams per page. |
| `search` | string | empty | Case-insensitive substring match against the team name. |
| `status` | `active` or `disabled` | `active` | Filters teams by team status. |

Example:

```http
GET /teams?page=2&limit=20&search=acme&status=active
```

When query parameters are omitted, the endpoint returns the first 20 active teams for the authenticated user.

#### Payload

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": [
    {
      "id": "string",
      "name": "Acme",
      "market_code": "string",
      "phone": "string",
      "address": "string",
      "website": "string",
      "status": "active",
      "user_role": "owner",
      "created_by": "string",
      "created_at": "2026-08-09T17:00:00Z",
      "updated_at": "2026-08-09T17:00:00Z"
    }
  ],
  "meta": {
    "pagination": {
      "page": 2,
      "limit": 20,
      "total": 37,
      "total_pages": 2
    }
  }
}
```

Teams are ordered by newest creation time first, with team id as the deterministic secondary ordering key.

#### Errors

Invalid `page`, `limit`, or `status` values return `400 Bad Request`. Other errors use the standard envelope in [README.md](README.md).

### `POST /teams`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "name": "string",
  "market_code": "string",
  "phone": "string",
  "address": "string",
  "website": "string"
}
```

#### Response — `201 Created`

```json
{
  "success": true,
  "data": {
    "id": "string",
    "name": "string",
    "market_code": "string",
    "phone": "string",
    "address": "string",
    "website": "string",
    "status": "string",
    "created_by": "string",
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /teams/invitations`

Alias of `GET /users/me/invitations`. Lists pending, unexpired invitations for the authenticated user's email address. The returned `id` can be passed to `POST /teams/invitations/:token/accept` or `POST /teams/invitations/:token/decline`; those routes continue to accept emailed tokens too.

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
      "team_name": "Example Team",
      "email": "user@example.com",
      "role": "member",
      "status": "pending",
      "invited_by": "string",
      "expires_at": "2026-08-16T17:00:00Z",
      "created_at": "2026-08-09T17:00:00Z",
      "updated_at": "2026-08-09T17:00:00Z"
    }
  ]
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /teams/invitations/:token`

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
    "email": "string",
    "role": "string",
    "status": "string",
    "invited_by": "string",
    "expires_at": "2026-08-09T17:00:00Z",
    "accepted_at": "2026-08-09T17:00:00Z",
    "declined_at": "2026-08-09T17:00:00Z",
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z",
    "token": "string"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /teams/invitations/:token/accept`

The `:token` path segment may be either the emailed invitation token or a pending invitation `id` returned by `GET /teams/invitations`.

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
    "email": "string",
    "role": "string",
    "status": "string",
    "invited_by": "string",
    "expires_at": "2026-08-09T17:00:00Z",
    "accepted_at": "2026-08-09T17:00:00Z",
    "declined_at": "2026-08-09T17:00:00Z",
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z",
    "token": "string"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /teams/invitations/:token/decline`

The `:token` path segment may be either the emailed invitation token or a pending invitation `id` returned by `GET /teams/invitations`.

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
    "email": "string",
    "role": "string",
    "status": "string",
    "invited_by": "string",
    "expires_at": "2026-08-09T17:00:00Z",
    "accepted_at": "2026-08-09T17:00:00Z",
    "declined_at": "2026-08-09T17:00:00Z",
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z",
    "token": "string"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /teams/:team_id`

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
    "name": "string",
    "market_code": "string",
    "phone": "string",
    "address": "string",
    "website": "string",
    "status": "string",
    "created_by": "string",
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `PATCH /teams/:team_id`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "name": "string"
}
```

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "id": "string",
    "name": "string",
    "market_code": "string",
    "phone": "string",
    "address": "string",
    "website": "string",
    "status": "string",
    "created_by": "string",
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `DELETE /teams/:team_id`

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
    "name": "string",
    "market_code": "string",
    "phone": "string",
    "address": "string",
    "website": "string",
    "status": "string",
    "created_by": "string",
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /teams/:team_id/members`

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
      "team_id": "string",
      "user_id": "string",
      "user": {
        "id": "string",
        "name": "string",
        "email": "user@example.com"
      },
      "role": "string",
      "status": "string",
      "created_at": "2026-08-09T17:00:00Z",
      "updated_at": "2026-08-09T17:00:00Z"
    }
  ]
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /teams/:team_id/members/invite`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "email": "string",
  "role": "string"
}
```

#### Response — `201 Created`

```json
{
  "success": true,
  "data": {
    "id": "string",
    "team_id": "string",
    "email": "string",
    "role": "string",
    "status": "string",
    "invited_by": "string",
    "expires_at": "2026-08-09T17:00:00Z",
    "accepted_at": "2026-08-09T17:00:00Z",
    "declined_at": "2026-08-09T17:00:00Z",
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z",
    "token": "string"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /teams/:team_id/invitations`

Lists outstanding pending, unexpired invitations for a team so team admins can render and manage invite rows.

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
      "email": "user@example.com",
      "role": "member",
      "status": "pending",
      "invited_by": "string",
      "expires_at": "2026-08-16T17:00:00Z",
      "created_at": "2026-08-09T17:00:00Z",
      "updated_at": "2026-08-09T17:00:00Z"
    }
  ]
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `DELETE /teams/:team_id/invitations/:invitation_id`

Revokes/cancels a pending team invitation.

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
    "team_name": "Example Team",
    "email": "user@example.com",
    "role": "member",
    "status": "revoked",
    "invited_by": "string",
    "expires_at": "2026-08-16T17:00:00Z",
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `DELETE /teams/:team_id/members/leave`

- Session: required.
- CSRF: required for browser requests.

#### Payload

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "left": true
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `DELETE /teams/:team_id/members/:user_id`

- Session: required.
- CSRF: required for browser requests.

#### Payload

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "removed": true
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `PATCH /teams/:team_id/members/:user_id`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "role": "string"
}
```

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "team_id": "string",
    "user_id": "string",
    "role": "string",
    "status": "string",
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

## Audit Events

### `GET /teams/:team_id/audit-events`

- Session: required.
- CSRF: not required.

#### Payload

Query parameters: `before`, `limit`.

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "events": [
      {
        "id": "string",
        "team_id": "string",
        "actor_type": "string",
        "actor_user_id": "string",
        "actor_session_id": "string",
        "actor_token_id": "string",
        "action": "string",
        "resource_type": "string",
        "resource_id": "string",
        "outcome": "string",
        "metadata": {},
        "request_id": "string",
        "ip_address": "string",
        "user_agent": "string",
        "created_at": "2026-08-09T17:00:00Z"
      }
    ],
    "next_cursor": "string"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.