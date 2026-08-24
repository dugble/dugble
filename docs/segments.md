# Segments

Customer dashboard HTTP contracts. Payloads and responses are generated from the public Go request and response types.

## Segment

### `POST /segments`

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
    "created_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /segments`

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
      "created_at": "2026-08-09T17:00:00Z"
    }
  ]
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /segments/:segment_id`

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
    "created_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /segments/:segment_id/contacts`

- Session: required.
- CSRF: not required.

#### Payload

Query parameters: `limit`, `offset`. `limit` defaults to `50` and is capped at `100`; negative `offset` values are treated as `0`.

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": [
    {
      "id": "string",
      "team_id": "string",
      "email": "string",
      "first_name": "string",
      "last_name": "string",
      "unsubscribed": true,
      "created_at": "2026-08-09T17:00:00Z",
      "updated_at": "2026-08-09T17:00:00Z"
    }
  ]
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `DELETE /segments/:segment_id`

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
    "created_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

## Topic

### `POST /topics`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "name": "string",
  "description": "string",
  "default_subscription": "string",
  "visibility": "string"
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

### `GET /topics`

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
    "has_more": true,
    "data": [
      {
        "object": "string",
        "id": "string",
        "name": "string",
        "description": "string",
        "default_subscription": "string",
        "visibility": "string",
        "created_at": "2026-08-09T17:00:00Z"
      }
    ]
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /topics/:topic_id`

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
    "name": "string",
    "description": "string",
    "default_subscription": "string",
    "visibility": "string",
    "created_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `PATCH /topics/:topic_id`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "name": "string",
  "description": "string",
  "visibility": "string"
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

### `DELETE /topics/:topic_id`

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
