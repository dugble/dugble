# Sender IDs API

Customer dashboard API for managing SMS Sender IDs.

> **Note:** This document describes the current API contract. It intentionally does not document a usage-count field because the current Sender ID resource does not expose one.

## Sender ID object

A Sender ID represents the sender identity used for SMS delivery.

```json
{
  "id": "string",
  "team_id": "string",
  "name": "string",
  "country_code": "string",
  "purpose": "string",
  "status": "string",
  "provider": "string",
  "rejection_reason": "string",
  "approved_at": "2026-08-09T17:00:00Z",
  "rejected_at": "2026-08-09T17:00:00Z",
  "suspended_at": "2026-08-09T17:00:00Z",
  "created_by": "string",
  "created_at": "2026-08-09T17:00:00Z",
  "updated_at": "2026-08-09T17:00:00Z"
}
```

### Fields

| Field | Description |
| --- | --- |
| `id` | Unique Sender ID identifier |
| `team_id` | Team that owns the Sender ID |
| `name` | Sender ID/name presented for SMS sending |
| `country_code` | Country associated with the Sender ID |
| `purpose` | Declared purpose for the Sender ID |
| `status` | Current Sender ID status |
| `provider` | SMS provider associated with the Sender ID |
| `rejection_reason` | Reason supplied when a Sender ID is rejected, when applicable |
| `approved_at` | Approval timestamp, when applicable |
| `rejected_at` | Rejection timestamp, when applicable |
| `suspended_at` | Suspension timestamp, when applicable |
| `created_by` | User that created the Sender ID |
| `created_at` | Creation timestamp |
| `updated_at` | Last update timestamp |

## List Sender IDs

### `GET /sender-ids`

Returns Sender IDs belonging to the current team.

- Session: required
- CSRF: not required
- Pagination: **not currently supported**

#### Request

No JSON request body or pagination parameters are currently defined.

#### Response — `200 OK`

```json
{
  "success": true,
  "data": [
    {
      "id": "string",
      "team_id": "string",
      "name": "Example",
      "country_code": "GH",
      "purpose": "transactional",
      "status": "pending",
      "provider": "string",
      "rejection_reason": "",
      "approved_at": "2026-08-09T17:00:00Z",
      "rejected_at": "2026-08-09T17:00:00Z",
      "suspended_at": "2026-08-09T17:00:00Z",
      "created_by": "string",
      "created_at": "2026-08-09T17:00:00Z",
      "updated_at": "2026-08-09T17:00:00Z"
    }
  ]
}
```

## Create a Sender ID

### `POST /sender-ids`

Creates a Sender ID request for the current team.

- Session: required
- CSRF: required for browser requests

#### Request body

```json
{
  "name": "Example",
  "country_code": "GH",
  "purpose": "transactional",
  "provider": "string"
}
```

#### Response — `201 Created`

```json
{
  "success": true,
  "data": {
    "id": "string",
    "team_id": "string",
    "name": "Example",
    "country_code": "GH",
    "purpose": "transactional",
    "status": "pending",
    "provider": "string",
    "rejection_reason": "",
    "approved_at": "2026-08-09T17:00:00Z",
    "rejected_at": "2026-08-09T17:00:00Z",
    "suspended_at": "2026-08-09T17:00:00Z",
    "created_by": "string",
    "created_at": "2026-08-09T17:00:00Z",
    "updated_at": "2026-08-09T17:00:00Z"
  }
}
```

## Get a Sender ID

### `GET /sender-ids/:sender_id`

Returns a single Sender ID belonging to the current team.

- Session: required
- CSRF: not required

#### Request

No JSON request body.

#### Response — `200 OK`

The response is a Sender ID object using the schema described above.

## Delete a Sender ID

### `DELETE /sender-ids/:sender_id`

Deletes/removes the specified Sender ID according to the current backend lifecycle rules.

- Session: required
- CSRF: required for browser requests

#### Request

No JSON request body.

#### Response — `200 OK`

The response contains the resulting Sender ID object using the schema described above.

## Status and lifecycle

The Sender ID resource exposes lifecycle timestamps for approval, rejection, and suspension. The exact set of valid `status` values is defined by the backend implementation and should be treated as an enum rather than free-form display text.

Frontend code should use the returned status and timestamps to render the current state instead of inferring state from the presence of a single timestamp.

## Usage counts

The current Sender ID API **does not expose a usage-count field**.

Do not infer usage from the Sender ID object's message count, timestamps, or status. If usage/quota information is added later, the API should define:

- what messages are counted;
- the time window, if any;
- whether the count is sent, delivered, or accepted messages;
- whether the value is a current-period usage or lifetime total; and
- whether the count is per Sender ID or per team.

Until that contract exists, the frontend should not display a Sender ID usage count based on this endpoint.

## Pagination

`GET /sender-ids` currently returns a plain array and does not document `limit`, `offset`, `after`, or `before` parameters.

This is intentional in the current contract. If Sender IDs need pagination as the dataset grows, pagination should be added as an explicit API change and documented here before the frontend depends on it.

## Errors

Sender ID endpoints use the standard API error envelope documented in `docs/README.md`. Expected errors include authentication, permission, validation, resource-not-found, conflict, rate-limit, and service failures according to the current API implementation.
