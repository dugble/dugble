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

Lists pending, unexpired team invitations for the authenticated user's email address. Use the returned invitation `id` with the user invitation accept or decline endpoints below.

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

### `POST /users/me/invitations/:invitation_id/accept`

Accepts a pending team invitation belonging to the authenticated user's email address. `:invitation_id` must be the UUID returned by `GET /users/me/invitations`.

On success, the invitation is marked accepted and the authenticated user is added to the team with the role carried by the invitation.

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
    "status": "accepted",
    "invited_by": "string",
    "expires_at": "2026-08-16T17:00:00Z",
    "accepted_at": "2026-08-09T17:00:00Z",
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Invalid invitation IDs, expired or unavailable invitations, email mismatches, and existing team membership are rejected.

### `POST /users/me/invitations/:invitation_id/decline`

Declines a pending team invitation belonging to the authenticated user's email address. `:invitation_id` must be the UUID returned by `GET /users/me/invitations`.

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
    "status": "declined",
    "invited_by": "string",
    "expires_at": "2026-08-16T17:00:00Z",
    "declined_at": "2026-08-09T17:00:00Z",
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Invalid invitation IDs, expired or unavailable invitations, and email mismatches are rejected.

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

Starts or replaces a pending email change. The current verified email remains the account email until the new address is verified.

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "email": "new@example.com",
  "current_password": "current-password"
}
```

The current password must be valid, the new email must differ from the current email, and the new email must not already belong to another user. A verification link valid for 24 hours is sent to the pending address.

#### Response — `202 Accepted`

```json
{
  "success": true,
  "data": {
    "email": "old@example.com",
    "pending_email": "new@example.com",
    "verification_expires_at": "2026-08-17T20:00:00Z"
  }
}
```

Calling this endpoint again replaces the pending email and invalidates the previous email-change verification token.

#### Errors

Invalid email or password input returns `400 Bad Request`. An incorrect current password returns `401 Unauthorized`. An email already in use returns `409 Conflict`.

### `POST /users/email/verify`

Verifies the pending email and commits the change. The token is single-use and must match the authenticated user's current pending email-change request.

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "token": "verification-token"
}
```

On success, the pending email becomes the account email, remains verified, the pending request is deleted, the token is consumed, the credential version is advanced, and all existing sessions are revoked. The old and new addresses receive an email-change security notification. The client should require a fresh sign-in after receiving the response.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "id": "string",
    "email": "new@example.com",
    "email_verified": true,
    "name": "string",
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-16T20:00:00Z"
  }
}
```

#### Errors

Missing, invalid, expired, used, or superseded verification tokens return `400 Bad Request`. If the pending address became unavailable before verification, the endpoint returns `409 Conflict`.

### `POST /users/email/resend`

Rotates the verification token for the authenticated user's pending email change and sends a new verification link. The pending address itself is not changed.

- Session: required.
- CSRF: required for browser requests.

#### Payload

No JSON request body.

#### Response — `202 Accepted`

```json
{
  "success": true,
  "data": {
    "email": "old@example.com",
    "pending_email": "new@example.com",
    "verification_expires_at": "2026-08-17T20:00:00Z"
  }
}
```

#### Errors

If there is no unexpired pending email change, the endpoint returns `400 Bad Request`. If the pending address has become unavailable, it returns `409 Conflict`.
