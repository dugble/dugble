# Authentication

Customer dashboard HTTP contracts. Payloads and responses are generated from the public Go request and response types.

## CSRF token

### `GET /csrf`

- Session: not required.
- CSRF: not required.

#### Payload

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "csrf_token": "string"
  }
}
```


## Auth

### `POST /auth/register`

- Session: not required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "email": "string",
  "name": "string",
  "password": "string"
}
```

#### Response — `201 Created`

```json
{
  "success": true,
  "data": {
    "user": {
      "id": "string",
      "email": "string",
      "email_verified": true,
      "name": "string",
      "created_at": "2026-08-09T17:00:00Z",
      "updated_at": "2026-08-09T17:00:00Z"
    }
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /auth/login`

- Session: not required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "email": "string",
  "password": "string"
}
```

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "user": {
      "id": "string",
      "email": "string",
      "email_verified": true,
      "name": "string",
      "created_at": "2026-08-09T17:00:00Z",
      "updated_at": "2026-08-09T17:00:00Z"
    },
    "mfa_required": true,
    "challenge_token": "string",
    "methods": [
      "string"
    ]
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /auth/login/mfa/totp`

- Session: not required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "challenge_token": "string",
  "code": "string"
}
```

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "user": {
      "id": "string",
      "email": "string",
      "email_verified": true,
      "name": "string",
      "created_at": "2026-08-09T17:00:00Z",
      "updated_at": "2026-08-09T17:00:00Z"
    },
    "mfa_required": true,
    "challenge_token": "string",
    "methods": [
      "string"
    ]
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /auth/login/mfa/recovery`

- Session: not required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "challenge_token": "string",
  "code": "string"
}
```

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "user": {
      "id": "string",
      "email": "string",
      "email_verified": true,
      "name": "string",
      "created_at": "2026-08-09T17:00:00Z",
      "updated_at": "2026-08-09T17:00:00Z"
    },
    "mfa_required": true,
    "challenge_token": "string",
    "methods": [
      "string"
    ]
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /auth/email/verify`

- Session: not required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "email": "string",
  "token": "string"
}
```

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "email_verified": true
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /auth/email/resend`

- Session: not required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "email": "string"
}
```

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "sent": true
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /auth/password/forgot`

- Session: not required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "email": "string"
}
```

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "sent": true
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /auth/password/reset`

- Session: not required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "email": "string",
  "token": "string",
  "password": "string"
}
```

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "password_reset": true
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /auth/user`

- Session: required.
- CSRF: not required.

#### Payload

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "user": {
      "id": "string",
      "email": "string",
      "email_verified": true,
      "name": "string",
      "created_at": "2026-08-09T17:00:00Z",
      "updated_at": "2026-08-09T17:00:00Z"
    }
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /auth/logout`

- Session: required.
- CSRF: required for browser requests.

#### Payload

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "logged_out": true
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

## Mfa

### `GET /auth/mfa`

- Session: required.
- CSRF: not required.

#### Payload

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "enabled": true
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /auth/mfa/totp/enroll`

- Session: required.
- CSRF: required for browser requests.

#### Payload

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "secret": "string",
    "uri": "string"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /auth/mfa/totp/confirm`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "code": "string"
}
```

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "recovery_codes": [
      "string"
    ]
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /auth/mfa/verify`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "code": "string"
}
```

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "verified": true
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /auth/mfa/recovery`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "code": "string"
}
```

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "verified": true
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `DELETE /auth/mfa`

- Session: required.
- CSRF: required for browser requests.

#### Payload

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "enabled": false
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

## Session

### `GET /sessions`

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
      "user_agent": "string",
      "ip_address": "string",
      "expires_at": "2026-08-09T17:00:00Z",
      "revoked_at": "2026-08-09T17:00:00Z",
      "created_at": "2026-08-09T17:00:00Z",
      "last_seen_at": "2026-08-09T17:00:00Z",
      "authentication_method": "string",
      "assurance_level": "string",
      "authenticated_at": "2026-08-09T17:00:00Z",
      "mfa_completed_at": "2026-08-09T17:00:00Z"
    }
  ]
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /sessions/:id`

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
      "user_agent": "string",
      "ip_address": "string",
      "expires_at": "2026-08-09T17:00:00Z",
      "revoked_at": "2026-08-09T17:00:00Z",
      "created_at": "2026-08-09T17:00:00Z",
      "last_seen_at": "2026-08-09T17:00:00Z",
      "authentication_method": "string",
      "assurance_level": "string",
      "authenticated_at": "2026-08-09T17:00:00Z",
      "mfa_completed_at": "2026-08-09T17:00:00Z"
    }
  ]
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `DELETE /sessions/others`

- Session: required.
- CSRF: required for browser requests.

#### Payload

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "revoked": true
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `DELETE /sessions/:id`

- Session: required.
- CSRF: required for browser requests.

#### Payload

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "revoked": true
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `DELETE /sessions`

- Session: required.
- CSRF: required for browser requests.

#### Payload

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "revoked": true
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.
