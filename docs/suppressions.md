# Suppressions

Customer dashboard HTTP contracts. Payloads and responses are generated from the public Go request and response types.

## Suppression

### `POST /suppressions/batch/add`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "emails": [
    "string"
  ]
}
```

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "data": [
      {
        "object": "string",
        "id": "string"
      }
    ]
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /suppressions/batch/remove`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "emails": [
    "string"
  ],
  "ids": [
    "string"
  ]
}
```

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "data": [
      {
        "object": "string",
        "id": "string",
        "deleted": true
      }
    ]
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /suppressions`

- Session: required.
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
    "object": "string",
    "id": "string"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /suppressions`

- Session: required.
- CSRF: not required.

#### Payload

Query parameters: `after`, `before`, `limit`, `origin`.

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
        "email": "string",
        "origin": "string",
        "source_id": "string",
        "created_at": "2026-08-09T17:00:00Z"
      }
    ]
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /suppressions/:suppression`

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
    "email": "string",
    "origin": "string",
    "source_id": "string",
    "created_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `DELETE /suppressions/:suppression`

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
