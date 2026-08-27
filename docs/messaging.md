# Messaging

Customer dashboard HTTP contracts. Payloads and responses are generated from the public Go request and response types.

## Message Templates

### Template categories

Message templates have a `category` field. Valid values are `otp`, `welcome`, `receipt`, `alert`, `notification`, and `custom`.

The category is persisted on the template and is used by the template editor to determine which category-specific merge variables are available.

### `POST /templates`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "name": "string",
  "category": "otp",
  "html": "string",
  "alias": "string",
  "from": "string",
  "subject": "string",
  "reply_to": "string",
  "text": "string",
  "variables": [
    {
      "key": "string",
      "type": "string",
      "fallback_value": "string"
    }
  ]
}
```

`category` must be one of `otp`, `welcome`, `receipt`, `alert`, `notification`, or `custom`.

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

### `GET /templates`

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
    "data": [
      {
        "id": "string",
        "name": "string",
        "category": "otp",
        "status": "string",
        "published_at": "2026-08-09T17:00:00Z",
        "created_at": "2026-08-09T17:00:00Z",
        "updated_at": "2026-08-09T17:00:00Z",
        "alias": "string"
      }
    ],
    "has_more": true
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /templates/:template`

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
    "current_version_id": "string",
    "alias": "string",
    "name": "string",
    "category": "otp",
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z",
    "status": "string",
    "published_at": "2026-08-09T17:00:00Z",
    "from": "string",
    "subject": "string",
    "reply_to": [
      "string"
    ],
    "html": "string",
    "text": "string",
    "variables": [
      {
        "id": "string",
        "key": "string",
        "type": "string",
        "fallback_value": "string",
        "created_at": "2026-08-09T17:00:00Z",
        "updated_at": "2026-08-09T17:00:00Z"
      }
    ],
    "has_unpublished_versions": true
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `PATCH /templates/:template`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "name": "string",
  "category": "otp",
  "html": "string",
  "alias": "string",
  "from": "string",
  "subject": "string",
  "reply_to": "string",
  "text": "string",
  "variables": [
    {
      "key": "string",
      "type": "string",
      "fallback_value": "string"
    }
  ]
}
```

`category` is optional on update. If supplied, it must be one of `otp`, `welcome`, `receipt`, `alert`, `notification`, or `custom`.

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

### `DELETE /templates/:template`

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

### `POST /templates/:template/publish`

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
    "id": "string"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /templates/:template/duplicate`

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
    "id": "string"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /templates/:template/versions`

- Session: required.
- CSRF: not required.

#### Payload

Query parameters: `limit`, `offset`.

No JSON request body.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": [
    {
      "id": "string",
      "team_id": "string",
      "template_id": "string",
      "version_number": 0,
      "from_email": "string",
      "from_name": "string",
      "reply_to_email": "string",
      "subject": "string",
      "html": "string",
      "text": "string",
      "variables": [
        {
          "key": "string",
          "type": "string",
          "fallback_value": "string"
        }
      ],
      "based_on_version_id": "string",
      "change_note": "string",
      "created_at": "2026-08-09T17:00:00Z"
    }
  ]
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /templates/:template/versions/:version_id`

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
    "template_id": "string",
    "version_number": 0,
    "from_email": "string",
    "from_name": "string",
    "reply_to_email": "string",
    "subject": "string",
    "html": "string",
    "text": "string",
    "variables": [
      {
        "key": "string",
        "type": "string",
        "fallback_value": "string"
      }
    ],
    "based_on_version_id": "string",
    "change_note": "string",
    "created_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /templates/:template/versions/:version_id/revert`

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
    "alias": "string",
    "current_version_id": "string",
    "published_version_id": "string",
    "published_at": "2026-08-09T17:00:00Z",
    "has_unpublished_changes": true,
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /templates/:template/preview`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "version_id": "string",
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

### `POST /templates/:template/test-send`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "to": "string",
  "version_id": "string",
  "variables": {}
}
```

#### Response — `202 Accepted`

```json
{
  "success": true,
  "data": "string"
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

## Sender Ids

### `GET /sender-ids`

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
      "id": "string"
    }
  ]
}
```
