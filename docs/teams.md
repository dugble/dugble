# Teams

Customer dashboard HTTP contracts. Payloads and responses are generated from the public Go request and response types.

## Team

### `GET /teams`

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
  ]
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

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

## Team Tokens

### `GET /team-tokens`

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
      "token_prefix": "string",
      "permissions": [
        "string"
      ],
      "created_by": "string",
      "expires_at": "2026-08-09T17:00:00Z",
      "revoked_at": "2026-08-09T17:00:00Z",
      "last_used_at": "2026-08-09T17:00:00Z",
      "created_at": "2026-08-09T17:00:00Z",
      "updated_at": "2026-08-09T17:00:00Z"
    }
  ]
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /team-tokens`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "name": "string",
  "permissions": [
    "string"
  ],
  "expires_at": "2026-08-09T17:00:00Z"
}
```

#### Response — `201 Created`

```json
{
  "success": true,
  "data": {
    "secret": "string"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `PATCH /team-tokens/:token_id`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "name": "string",
  "permissions": [
    "string"
  ],
  "expires_at": "2026-08-09T17:00:00Z"
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
    "token_prefix": "string",
    "permissions": [
      "string"
    ],
    "created_by": "string",
    "expires_at": "2026-08-09T17:00:00Z",
    "revoked_at": "2026-08-09T17:00:00Z",
    "last_used_at": "2026-08-09T17:00:00Z",
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `DELETE /team-tokens/:token_id`

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
    "token_prefix": "string",
    "permissions": [
      "string"
    ],
    "created_by": "string",
    "expires_at": "2026-08-09T17:00:00Z",
    "revoked_at": "2026-08-09T17:00:00Z",
    "last_used_at": "2026-08-09T17:00:00Z",
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
