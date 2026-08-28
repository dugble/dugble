# Templates API

API reference for reusable message templates.

This document reflects the current `messagetemplate` module request/response types and routes. It deliberately avoids documenting fields or semantics that are not present in the implementation.

## Categories

The API defines these template categories:

- `otp`
- `welcome`
- `receipt`
- `alert`
- `notification`
- `custom`

The `category` field is required when creating a template and may be supplied when updating one.

## Template resource

A template resource has this shape:

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

Nullable fields may be omitted or returned as `null` according to the API serializer.

## Create

### `POST /templates`

Requires the `templates:write` permission.

### Request body

```json
{
  "name": "Welcome email",
  "html": "<h1>Hello {{first_name}}</h1>",
  "alias": "welcome-email",
  "category": "welcome",
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

The API request accepts `reply_to` as either a single string or an array of strings. Variables use `key`, `type`, and an optional `fallback_value`.

The implementation currently defines variable types `string` and `number`.

The handler returns the service result through the standard `200 OK` response helper; do not assume `201 Created` for this endpoint.

## List

### `GET /templates`

Requires the `templates:read` permission.

### Query parameters

The HTTP handler accepts:

- `limit`
- `after`
- `before`

The template API therefore uses cursor-style pagination at the HTTP boundary.

### Response

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
        "status": "string",
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

## Retrieve

### `GET /templates/:template`

Requires the `templates:read` permission.

Returns the full template resource, including the current version's rendered content fields and variables.

## Update

### `PATCH /templates/:template`

Requires the `templates:write` permission.

The current public request shape allows these optional fields:

```json
{
  "name": "Welcome email",
  "html": "<h1>Hello {{first_name}}</h1>",
  "alias": "welcome-email",
  "category": "welcome",
  "from": "hello@example.com",
  "subject": "Welcome, {{first_name}}",
  "reply_to": ["support@example.com"],
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

## Delete

### `DELETE /templates/:template`

Requires the `templates:write` permission.

Response shape:

```json
{
  "object": "template",
  "id": "string",
  "deleted": true
}
```

## Publish

### `POST /templates/:template/publish`

Requires the `templates:write` permission.

The route does not expose a request body schema in the HTTP handler. Publishing is performed against the template selected by the route.

## Duplicate

### `POST /templates/:template/duplicate`

Requires the `templates:write` permission.

The current route handler does not decode a request body, so do not rely on `name` or `alias` override fields being accepted by this endpoint.

## Versions

### `GET /templates/:template/versions`

Requires the `templates:read` permission.

The handler accepts:

- `limit`
- `offset`

This endpoint is offset-based, unlike the top-level template list endpoint.

### `GET /templates/:template/versions/:version_id`

Requires the `templates:read` permission.

Returns a specific template version.

### `POST /templates/:template/versions/:version_id/revert`

Requires the `templates:write` permission.

Reverts the template to the selected version.

## Preview

### `POST /templates/:template/preview`

Requires the `templates:read` permission.

The request body is optional at the HTTP decoding layer:

```json
{
  "version_id": "string",
  "variables": {
    "first_name": "Ada"
  }
}
```

Response shape:

```json
{
  "template_id": "string",
  "version_id": "string",
  "subject": "string",
  "html": "string",
  "text": "string",
  "from_email": "string",
  "from_name": "string",
  "reply_to": "string"
}
```

Nullable response fields may be omitted or returned as `null`.

## Test send

### `POST /templates/:template/test-send`

Requires the `templates:write` permission.

Request body:

```json
{
  "to": "test@example.com",
  "version_id": "string",
  "variables": {
    "first_name": "Ada"
  }
}
```

The handler returns `202 Accepted` through the standard accepted-response helper.

## Template usage

The current template model does **not** contain a `sent_last_30d` field. Do not treat `sent_last_30d` as part of the current template resource contract until an API response type explicitly exposes it.

The backend work that links sent emails to templates may support calculating such an aggregate, but that is separate from the currently documented template resource.
