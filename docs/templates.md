# Templates API

Customer dashboard API for managing reusable message templates.

> **Note:** This document describes the current API contract. Request and response examples should be kept in sync with the public Go request/response types.

## Template categories

Every template has a `category` used by the dashboard for template classification and variable-mapping logic.

Allowed values:

| Category | Intended use |
| --- | --- |
| `otp` | One-time passwords and verification codes |
| `welcome` | Welcome and onboarding messages |
| `receipt` | Receipts and transaction confirmations |
| `alert` | Important alerts |
| `notification` | General notifications |
| `custom` | User-defined/general-purpose templates |

## Template object

A template contains reusable message metadata and content. The current resource includes fields such as:

```json
{
  "object": "template",
  "id": "string",
  "current_version_id": "string",
  "alias": "string",
  "name": "string",
  "category": "custom",
  "created_at": "2026-08-09T17:00:00Z",
  "updated_at": "2026-08-09T17:00:00Z",
  "status": "string",
  "published_at": "2026-08-09T17:00:00Z",
  "from": "string",
  "subject": "string",
  "reply_to": ["string"],
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
  "has_unpublished_versions": false
}
```

## Create a template

### `POST /templates`

- Session: required
- CSRF: required for browser requests

#### Request body

```json
{
  "name": "Welcome email",
  "category": "welcome",
  "html": "<h1>Hello {{first_name}}</h1>",
  "alias": "welcome-email",
  "from": "hello@example.com",
  "subject": "Welcome, {{first_name}}",
  "reply_to": "support@example.com",
  "text": "Hello {{first_name}}",
  "variables": [
    {
      "key": "first_name",
      "type": "string",
      "fallback_value": "there"
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
    "object": "template",
    "id": "string",
    "category": "welcome"
  }
}
```

## List templates

### `GET /templates`

- Session: required
- CSRF: not required

#### Query parameters

| Parameter | Description |
| --- | --- |
| `after` | Cursor for the next page |
| `before` | Cursor for the previous page |
| `limit` | Maximum number of templates to return |

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "object": "list",
    "data": [
      {
        "id": "string",
        "name": "Welcome email",
        "category": "welcome",
        "status": "published",
        "published_at": "2026-08-09T17:00:00Z",
        "created_at": "2026-08-09T17:00:00Z",
        "updated_at": "2026-08-09T17:00:00Z",
        "alias": "welcome-email"
      }
    ],
    "has_more": false
  }
}
```

## Get a template

### `GET /templates/:template`

- Session: required
- CSRF: not required

Returns the template, including its current content and variables.

## Update a template

### `PATCH /templates/:template`

- Session: required
- CSRF: required for browser requests

The request uses the same editable fields as template creation. `category`, when provided, must be one of the supported category values.

```json
{
  "name": "Welcome email",
  "category": "welcome",
  "html": "<h1>Hello {{first_name}}</h1>",
  "alias": "welcome-email",
  "from": "hello@example.com",
  "subject": "Welcome, {{first_name}}",
  "reply_to": "support@example.com",
  "text": "Hello {{first_name}}",
  "variables": [
    {
      "key": "first_name",
      "type": "string",
      "fallback_value": "there"
    }
  ]
}
```

## Delete a template

### `DELETE /templates/:template`

- Session: required
- CSRF: required for browser requests

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "object": "template",
    "id": "string",
    "deleted": true
  }
}
```

## Publish a template

### `POST /templates/:template/publish`

Publishes the current template version.

- Session: required
- CSRF: required for browser requests

## Duplicate a template

### `POST /templates/:template/duplicate`

Creates a copy of the template.

- Session: required
- CSRF: required for browser requests

The template's category is preserved when it is duplicated.

## Template versions

Templates are versioned. The version endpoints allow the dashboard to inspect and revert historical content.

### `GET /templates/:template/versions`

- Session: required
- CSRF: not required

Query parameters:

- `limit`
- `offset`

### `GET /templates/:template/versions/:version_id`

Returns a specific template version.

### `POST /templates/:template/versions/:version_id/revert`

Reverts the template to the selected version.

- Session: required
- CSRF: required for browser requests

## Preview

### `POST /templates/:template/preview`

Renders a template version with supplied variables without sending a message.

#### Request body

```json
{
  "version_id": "string",
  "variables": {
    "first_name": "Ada"
  }
}
```

#### Response — `200 OK`

```json
{
  "success": true,
  "data": {
    "template_id": "string",
    "version_id": "string",
    "subject": "Welcome, Ada",
    "html": "<h1>Hello Ada</h1>",
    "text": "Hello Ada",
    "from_email": "hello@example.com",
    "from_name": "",
    "reply_to": "support@example.com"
  }
}
```

## Test send

### `POST /templates/:template/test-send`

Sends a rendered test message using the selected version and variables.

- Session: required
- CSRF: required for browser requests

#### Request body

```json
{
  "to": "test@example.com",
  "version_id": "string",
  "variables": {
    "first_name": "Ada"
  }
}
```

#### Response — `202 Accepted`

```json
{
  "success": true,
  "data": "string"
}
```

## Template usage

`sent_last_30d` is a usage aggregate rather than a persisted template column. When exposed by the API, it represents the number of associated email messages sent during the preceding 30 days.

Frontend code should treat this field as optional until it is present in the endpoint's response schema. Do not expect a `sent_last_30d` database field.

## Errors

Template endpoints use the standard API error envelope documented in `docs/README.md`. Authentication, permission, validation, resource-not-found, conflict, rate-limit, and service errors use the status codes defined by the current API implementation.
