# Contacts

Customer dashboard HTTP contracts. Payloads and responses are generated from the public Go request and response types.

## Contact

### `POST /contacts`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "email": "string",
  "phone": "string",
  "sms_consent_status": "string",
  "sms_consent_source": "string",
  "first_name": "string",
  "last_name": "string",
  "unsubscribed": true,
  "properties": {}
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
    "phone": "string",
    "normalized_phone": "string",
    "phone_country": "string",
    "sms_consent_status": "string",
    "sms_consent_updated_at": "2026-08-09T17:00:00Z",
    "sms_consent_source": "string",
    "first_name": "string",
    "last_name": "string",
    "unsubscribed": true,
    "properties": {},
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /contacts`

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
      "email": "string",
      "phone": "string",
      "normalized_phone": "string",
      "phone_country": "string",
      "sms_consent_status": "string",
      "sms_consent_updated_at": "2026-08-09T17:00:00Z",
      "sms_consent_source": "string",
      "first_name": "string",
      "last_name": "string",
      "unsubscribed": true,
      "properties": {},
      "created_at": "2026-08-09T17:00:00Z",
      "updated_at": "2026-08-09T17:00:00Z"
    }
  ]
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /contacts/:contact_id/topics`

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
        "id": "string",
        "name": "string",
        "description": "string",
        "subscription": "string"
      }
    ]
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `PATCH /contacts/:contact_id/topics`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
"string"
```

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "id": "string"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /contacts/:contact_id/segments`

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
      "created_at": "2026-08-09T17:00:00Z",
      "assigned_at": "2026-08-09T17:00:00Z"
    }
  ]
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `POST /contacts/:contact_id/segments/:segment_id`

- Session: required.
- CSRF: required for browser requests.

#### Payload

No JSON request body.

#### Response — `201 Created`

```json
{
  "success": true,
  "data": {
    "id": "string",
    "team_id": "string",
    "name": "string",
    "created_at": "2026-08-09T17:00:00Z",
    "assigned_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `DELETE /contacts/:contact_id/segments/:segment_id`

- Session: required.
- CSRF: required for browser requests.

#### Payload

No JSON request body.

#### Response — `204 No Content`

No response body.

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /contacts/:contact_id`

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
    "phone": "string",
    "normalized_phone": "string",
    "phone_country": "string",
    "sms_consent_status": "string",
    "sms_consent_updated_at": "2026-08-09T17:00:00Z",
    "sms_consent_source": "string",
    "first_name": "string",
    "last_name": "string",
    "unsubscribed": true,
    "properties": {},
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `PATCH /contacts/:contact_id`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "email": "string",
  "phone": "string",
  "sms_consent_status": "string",
  "sms_consent_source": "string",
  "first_name": "string",
  "last_name": "string",
  "unsubscribed": true,
  "properties": {}
}
```

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "id": "string",
    "team_id": "string",
    "email": "string",
    "phone": "string",
    "normalized_phone": "string",
    "phone_country": "string",
    "sms_consent_status": "string",
    "sms_consent_updated_at": "2026-08-09T17:00:00Z",
    "sms_consent_source": "string",
    "first_name": "string",
    "last_name": "string",
    "unsubscribed": true,
    "properties": {},
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `DELETE /contacts/:contact_id`

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
    "phone": "string",
    "normalized_phone": "string",
    "phone_country": "string",
    "sms_consent_status": "string",
    "sms_consent_updated_at": "2026-08-09T17:00:00Z",
    "sms_consent_source": "string",
    "first_name": "string",
    "last_name": "string",
    "unsubscribed": true,
    "properties": {},
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

## Contactproperty

### `POST /contact-properties`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "key": "string",
  "type": "string",
  "fallback_value": "string"
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

### `GET /contact-properties`

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
        "key": "string",
        "type": "string",
        "fallback_value": "string",
        "created_at": "2026-08-09T17:00:00Z"
      }
    ]
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `GET /contact-properties/:property_id`

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
    "key": "string",
    "type": "string",
    "fallback_value": "string",
    "created_at": "2026-08-09T17:00:00Z"
  }
}
```

#### Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on validation, authentication, permission, resource existence, conflict, rate limiting, and service availability.

### `PATCH /contact-properties/:property_id`

- Session: required.
- CSRF: required for browser requests.

#### Payload

```json
{
  "fallback_value": "string"
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

### `DELETE /contact-properties/:property_id`

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
