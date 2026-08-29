# Sender IDs API

Dashboard-facing HTTP contract for managing team SMS sender IDs.

## Conventions

- All routes are scoped to the authenticated team.
- List and retrieve operations require sender-ID read permission. Create and delete operations require their corresponding sender-ID permissions.
- JSON responses use the standard `{ "success": true, "data": ... }` envelope.
- Sender-ID IDs are UUID strings and timestamps are RFC 3339 strings.
- The provider selected internally is not exposed in sender-ID responses.

## Sender ID object

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "team_id": "650e8400-e29b-41d4-a716-446655440000",
  "name": "Dugble",
  "country_code": "GH",
  "purpose": "Transactional notifications",
  "status": "pending",
  "rejection_reason": null,
  "approved_at": null,
  "rejected_at": null,
  "suspended_at": null,
  "created_by": "750e8400-e29b-41d4-a716-446655440000",
  "created_at": "2026-08-29T12:00:00Z",
  "updated_at": "2026-08-29T12:00:00Z"
}
```

Nullable fields are omitted when they do not have a value.

## List sender IDs

### `GET /sender-ids`

Returns all sender IDs belonging to the current team.

- Session: required.
- CSRF: not required.

#### Request body

None.

#### Response

`200 OK` with an array of [sender ID objects](#sender-id-object).

```json
{
  "success": true,
  "data": []
}
```

## Create a sender ID

### `POST /sender-ids`

- Session: required.
- CSRF: required for browser requests.

#### Request body

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `name` | string | Yes | Sender ID requested for outgoing SMS. |
| `country_code` | string | Yes | Destination country code. |
| `purpose` | string | Yes | Intended use of the sender ID. |
| `provider` | string | No | Preferred provider, when provider selection is supported. |

```json
{
  "name": "Dugble",
  "country_code": "GH",
  "purpose": "Transactional notifications"
}
```

#### Response

`201 Created` with the created [sender ID object](#sender-id-object).

## Retrieve a sender ID

### `GET /sender-ids/:sender_id`

Returns one sender ID belonging to the current team.

- Session: required.
- CSRF: not required.

#### Path parameters

| Parameter | Type | Description |
| --- | --- | --- |
| `sender_id` | string (UUID) | Sender ID to retrieve. |

#### Request body

None.

#### Response

`200 OK` with the requested [sender ID object](#sender-id-object).

## Delete a sender ID

### `DELETE /sender-ids/:sender_id`

Disables a sender ID belonging to the current team and returns its updated representation.

- Session: required.
- CSRF: required for browser requests.

#### Path parameters

| Parameter | Type | Description |
| --- | --- | --- |
| `sender_id` | string (UUID) | Sender ID to disable. |

#### Request body

None.

#### Response

`200 OK` with the updated [sender ID object](#sender-id-object).

## Errors

Errors use the standard envelope in [README.md](README.md). Expected statuses depend on request validation, authentication, permission, resource existence, and conflicts.
