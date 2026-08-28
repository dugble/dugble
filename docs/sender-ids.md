# Sender IDs API

API reference for SMS Sender IDs.

This document reflects the current `senderid` module request/response types and routes. It avoids inventing fields, pagination, or usage metrics that are not present in the implementation.

## Sender ID resource

The Sender ID model exposes:

```json
{
  "id": "string",
  "team_id": "string",
  "name": "string",
  "country_code": "string",
  "purpose": "string",
  "status": "pending",
  "rejection_reason": "string",
  "approved_at": "2026-08-09T17:00:00Z",
  "rejected_at": "2026-08-09T17:00:00Z",
  "suspended_at": "2026-08-09T17:00:00Z",
  "created_by": "string",
  "created_at": "2026-08-09T17:00:00Z",
  "updated_at": "2026-08-09T17:00:00Z"
}
```

`provider` exists internally on the model but is tagged `json:"-"`, so it is **not part of the public JSON response**. fileciteturn33file0

The public status values defined by the module are:

- `pending`
- `approved`
- `rejected`
- `suspended`
- `inactive`

## List

### `GET /sender-ids`

Requires the `sender_ids:read` permission.

Returns the Sender IDs for the current team.

The route currently accepts no pagination parameters and returns a plain array through the standard `200 OK` response helper. fileciteturn36file0 fileciteturn37file0

### Response data

```json
[
  {
    "id": "string",
    "team_id": "string",
    "name": "Example",
    "country_code": "GH",
    "purpose": "transactional",
    "status": "pending",
    "rejection_reason": null,
    "approved_at": null,
    "rejected_at": null,
    "suspended_at": null,
    "created_by": null,
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
]
```

The outer `success`/`data` envelope is added by the standard HTTP response helper.

## Create

### `POST /sender-ids`

Requires the `sender_ids:create` permission. The handler returns `201 Created`. fileciteturn36file0 fileciteturn37file0

### Request body

```json
{
  "name": "Example",
  "country_code": "GH",
  "purpose": "transactional",
  "provider": "moolre"
}
```

The request fields are:

- `name` — normalized and validated as a Sender ID name.
- `country_code` — required and must be an uppercase two-letter ISO 3166-1 alpha-2 code.
- `purpose` — required; maximum 500 characters.
- `provider` — optional; maximum 120 characters after trimming.

For Ghana (`GH`), the service forces the provider to Moolre. Supplying a different provider for Ghana is rejected. Supplying Moolre for a non-Ghana Sender ID is also rejected. fileciteturn38file0

### Response

The created Sender ID is returned using the standard `201 Created` response helper. fileciteturn36file0

## Retrieve

### `GET /sender-ids/:sender_id`

Requires the `sender_ids:read` permission. fileciteturn37file0

The `sender_id` path parameter must be a valid UUID. An invalid UUID produces a bad-request error; a UUID that is not found within the current team produces a not-found error. fileciteturn38file0

The response is the same public Sender ID resource described above.

## Delete

### `DELETE /sender-ids/:sender_id`

Requires the `sender_ids:delete` permission. fileciteturn37file0

This operation deactivates the Sender ID through the repository rather than exposing a separate update endpoint. The handler returns the resulting Sender ID with the standard `200 OK` response helper. fileciteturn36file0turn38file0

## Validation and conflicts

Sender ID creation validates the name, country code, purpose, and optional provider. The service also scopes all Sender ID operations to the current team's tenant context. fileciteturn38file0

Creating a Sender ID that already exists for the same team and country returns a conflict error (`Sender ID already exists for this team and country`). fileciteturn38file0

## Usage counts

The current public Sender ID resource has **no usage-count field**. The internal `provider` field is also deliberately excluded from JSON responses. fileciteturn33file0

The frontend should not infer a Sender ID usage count from this endpoint. If usage/quota information is added later, it should be introduced as an explicit API contract with a defined counting window and semantics.

## Pagination

`GET /sender-ids` currently has no `limit`, `offset`, `after`, or `before` parameters. The handler calls the service without pagination arguments and returns the complete service result. fileciteturn36file0

If pagination is introduced later, it should be added as an explicit API change rather than assumed by the frontend.

## Authentication and authorization

The Sender ID routes are protected by permission middleware:

- `GET /sender-ids` → `sender_ids:read`
- `POST /sender-ids` → `sender_ids:create`
- `GET /sender-ids/:sender_id` → `sender_ids:read`
- `DELETE /sender-ids/:sender_id` → `sender_ids:delete` fileciteturn37file0

Browser CSRF requirements are enforced by the application's HTTP middleware; this module's route registration itself defines the authorization permissions above.
