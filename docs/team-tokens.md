# Team Tokens

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

`permissions` must contain one or more of the following values:

```json
{
  "name": "string",
  "permissions": {
    "type": "array",
    "items": {
      "type": "string",
      "enum": [
        "team:read",
        "team:update",
        "team_members:read",
        "team_member:invite",
        "sender_ids:read",
        "sender_ids:create",
        "sender_ids:delete",
        "sender_domains:read",
        "sender_domains:create",
        "sender_domains:delete",
        "sms:read",
        "sms:send",
        "email:read",
        "email:send",
        "verify:read",
        "verify:send",
        "verify:check",
        "verify:manage",
        "contacts:read",
        "contacts:write",
        "contact_properties:read",
        "contact_properties:write",
        "topics:read",
        "topics:write",
        "segments:read",
        "segments:write",
        "suppressions:read",
        "suppressions:write",
        "broadcasts:read",
        "broadcasts:write",
        "broadcasts:send",
        "templates:read",
        "templates:write",
        "wallet:read",
        "wallet_ledger:read"
      ]
    }
  },
  "expires_at": "2026-08-09T17:00:00Z"
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
    "token_prefix": "dgb_team_xxxxxxxx",
    "permissions": [
      "team:read"
    ],
    "created_by": "string",
    "expires_at": "2026-11-07T17:00:00Z",
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z",
    "secret": "dgb_team_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
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
