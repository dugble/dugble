# Users

Customer dashboard HTTP contracts. Payloads and responses are generated from the public Go request and response types.

## User

### `GET /users/me`

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
    "email": "string",
    "email_verified": true,
    "name": "string",
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /users/me/invitations`

Lists pending, unexpired team invitations for the authenticated user's email address. The returned invitation `id` can be passed to `POST /teams/invitations/:token/accept` or `POST /teams/invitations/:token/decline`; those routes also continue to accept emailed invitation tokens.

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

### `PATCH /users/me`

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
    "email": "string",
    "email_verified": true,
    "name": "string",
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `DELETE /users/me`

- Session: required.
- CSRF: required for browser requests.

#### Payload

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "deleted": true
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `PATCH /users/password`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "password": "string"
}
```

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "id": "string",
    "email": "string",
    "email_verified": true,
    "name": "string",
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `PATCH /users/email`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "email": "string"
}
```

#### Email verification flow

The current implementation applies the new email immediately, sets `email_verified` to `false`, and sends email-change notifications to the old and new addresses. It does not create a pending email address or require confirmation before changing `email`.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "id": "string",
    "email": "string",
    "email_verified": false,
    "name": "string",
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.
