# Broadcasts

Customer dashboard HTTP contract for email broadcasts.

A broadcast owns the exact email content that it sends. Reusable message templates are separate resources: a client may copy template content into a broadcast, but the Broadcast API does not accept or persist `template_id` or `template_version_id`, and delivery never dereferences a message template.

## Broadcast resource

A Broadcast response contains the audience, message content, lifecycle state, delivery counters, and optimistic-concurrency revision:

```json
{
  "id": "b4d8a79c-5101-4cd5-861d-bd3ea049a036",
  "team_id": "d72bb62f-f8f0-46bc-a3a3-e86bd56d98f6",
  "name": "August product update",
  "status": "draft",
  "segment_id": "0f593c7a-167e-4fe0-aeb8-6be39078d0f0",
  "topic_id": "381a82e7-e37e-42de-b902-23e34cf89d42",
  "from_email": "hello@example.com",
  "from_name": "Dugble",
  "reply_to_email": "support@example.com",
  "subject": "What's new this month",
  "preview_text": "A quick look at what shipped.",
  "html": "<p>Hello {{{FIRST_NAME}}}</p>",
  "text": "Hello {{{FIRST_NAME}}}",
  "variable_bindings": {
    "FIRST_NAME": "there"
  },
  "audience_count": 0,
  "eligible_count": 0,
  "suppressed_count": 0,
  "queued_count": 0,
  "failed_count": 0,
  "revision": 1,
  "created_at": "2026-08-30T08:00:00Z",
  "updated_at": "2026-08-30T08:00:00Z"
}
```

Lifecycle timestamps (`scheduled_at`, `queued_at`, `sent_at`, and `canceled_at`) are included when applicable. Optional message fields are omitted when unset.

## `POST /broadcasts`

Creates a broadcast. By default the result is a `draft`.

```json
{
  "name": "August product update",
  "segment_id": "0f593c7a-167e-4fe0-aeb8-6be39078d0f0",
  "topic_id": null,
  "from_email": "hello@example.com",
  "from_name": "Dugble",
  "reply_to_email": "support@example.com",
  "subject": "What's new this month",
  "preview_text": "A quick look at what shipped.",
  "html": "<p>Hello {{{FIRST_NAME}}}</p>",
  "text": "Hello {{{FIRST_NAME}}}",
  "variable_bindings": {
    "FIRST_NAME": "there"
  }
}
```

`segment_id`, `subject`, and `html` are required. `name` is optional and defaults to `subject` when omitted or blank. `topic_id`, `from_email`, `from_name`, `reply_to_email`, `preview_text`, `text`, and `variable_bindings` are optional. When `from_email` is omitted, delivery may use the configured default sender.

Creation can also queue or schedule the broadcast in the same request. Set `send` to `true` to queue immediately:

```json
{
  "segment_id": "0f593c7a-167e-4fe0-aeb8-6be39078d0f0",
  "subject": "Send now",
  "html": "<p>Hello</p>",
  "send": true
}
```

To schedule instead, combine `send: true` with a future `scheduled_at` value:

```json
{
  "segment_id": "0f593c7a-167e-4fe0-aeb8-6be39078d0f0",
  "subject": "Scheduled update",
  "html": "<p>Hello</p>",
  "send": true,
  "scheduled_at": "2026-09-01T09:00:00Z"
}
```

Supplying `scheduled_at` without `send: true`, or supplying a timestamp that is not in the future, returns a bad-request error.

Successful creation returns `201 Created` with the Broadcast resource.

## `GET /broadcasts`

Lists broadcasts for the current team, newest first.

The endpoint accepts the standard `limit` and `offset` pagination parameters. The service defaults invalid or omitted limits to 50 and caps accepted limits at 100.

Returns `200 OK` with a JSON array of Broadcast resources.

## `GET /broadcasts/:broadcast`

Returns one Broadcast resource for the current team.

Returns `200 OK`, or `404 Not Found` when the broadcast does not exist in the team scope.

## `PATCH /broadcasts/:broadcast`

Updates a `draft` or `scheduled` broadcast. Content and audience fields are edited directly; no template snapshot is created.

Every update requires the current positive `revision` value:

```json
{
  "revision": 2,
  "name": "August product update",
  "segment_id": "0f593c7a-167e-4fe0-aeb8-6be39078d0f0",
  "topic_id": null,
  "from_email": "hello@example.com",
  "from_name": "Dugble",
  "reply_to_email": null,
  "subject": "What's new this month",
  "preview_text": "Updated preview text",
  "html": "<p>Updated content.</p>",
  "text": "Updated content.",
  "variable_bindings": {
    "FIRST_NAME": "there"
  }
}
```

The request is partial. For nullable fields (`topic_id`, `from_email`, `from_name`, `reply_to_email`, `preview_text`, and `text`), omission means “leave unchanged” and explicit JSON `null` means “clear this field”. `variable_bindings: null` is normalized to an empty object when that field is included.

A successful update increments `revision` and returns `200 OK`. A stale revision, or an attempt to edit a broadcast after it has entered `queued`, returns a conflict.

Updating a scheduled broadcast does not remove its existing schedule.

## `POST /broadcasts/:broadcast/send`

Queues a draft immediately when the body is empty or `scheduled_at` is omitted:

```json
{}
```

Schedules a draft, or reschedules an already scheduled broadcast, when `scheduled_at` is a future timestamp:

```json
{
  "scheduled_at": "2026-09-01T09:00:00Z"
}
```

Immediate send is only valid for a `draft`. A scheduled broadcast must be rescheduled with another future timestamp rather than sent immediately through this endpoint.

Returns `202 Accepted` with the updated Broadcast resource.

## `POST /broadcasts/:broadcast/cancel`

Cancels active scheduling or execution. No request body is required.

- `scheduled` -> `draft`: clears `scheduled_at` so the broadcast can be edited or scheduled again.
- `queued` -> `canceled`: stops remaining broadcast fanout work. Email already handed to delivery cannot be recalled.

Other states return a conflict. Successful cancellation returns `200 OK` with the updated Broadcast resource.

## `DELETE /broadcasts/:broadcast`

Soft-deletes a broadcast. Only `draft` and `canceled` broadcasts can be deleted.

Returns `200 OK` with the deleted Broadcast resource. Other lifecycle states return a conflict.

## `POST /broadcasts/:broadcast/duplicate`

Creates a new draft by copying the source broadcast's audience and exact message content.

```json
{
  "name": "August product update Copy"
}
```

`name` is optional. When omitted or blank, the new name defaults to `<source name> Copy`.

Returns `201 Created` with the new Broadcast resource.

## `POST /broadcasts/:broadcast/preview`

Renders the broadcast-owned subject, preview text, HTML, and text without performing a message-template lookup.

```json
{
  "variables": {
    "FIRST_NAME": "Ada"
  }
}
```

Broadcast `variable_bindings` are the base values and request `variables` override them. Placeholders use triple braces and identifier-style keys, for example `{{{FIRST_NAME}}}`. Values inserted into HTML are escaped; subject, preview text, and text-body substitutions are not HTML-escaped. A referenced variable with no value returns a bad-request error.

Example response:

```json
{
  "from_email": "hello@example.com",
  "from_name": "Dugble",
  "subject": "Hello Ada",
  "preview_text": "Welcome, Ada",
  "html": "<p>Hello Ada</p>",
  "text": "Hello Ada"
}
```

Returns `200 OK`.

## `GET /broadcasts/:broadcast/recipients`

Lists materialized recipient snapshots for a broadcast. Supports the standard `limit` and `offset` pagination parameters.

Recipient records include the snapshotted email/contact data, fanout status, optional exclusion reason, optional queued email message ID, and relevant timestamps.

Returns `200 OK` with a JSON array.

## `GET /broadcasts/:broadcast/exclusions`

Returns the materialized exclusion summary:

```json
{
  "object": "broadcast.exclusion_summary",
  "broadcast_id": "b4d8a79c-5101-4cd5-861d-bd3ea049a036",
  "total": 3,
  "reasons": {
    "global_unsubscribe": 1,
    "suppressed": 1,
    "topic_unsubscribed": 1
  }
}
```

Returns `200 OK`.

## `GET /broadcasts/:broadcast/analytics`

Returns aggregate delivery and engagement counts for the broadcast:

```json
{
  "object": "broadcast.analytics",
  "broadcast_id": "b4d8a79c-5101-4cd5-861d-bd3ea049a036",
  "audience": 100,
  "eligible": 92,
  "excluded": 8,
  "queued": 90,
  "delivered": 84,
  "bounced": 2,
  "complained": 1,
  "failed": 3,
  "opened": 48,
  "clicked": 17
}
```

Returns `200 OK`.

## Lifecycle

```text
draft -- send now --------------------> queued -- complete --> sent
  |                                       |
  | schedule                              +-- cancel -------> canceled
  v
scheduled -- due ---------------------> queued
  |
  +-- cancel schedule ----------------> draft
  |
  +-- reschedule ---------------------> scheduled
```

The supported statuses are `draft`, `scheduled`, `queued`, `sent`, `failed`, and `canceled`.

Message content is editable only while the broadcast is `draft` or `scheduled`. Once the broadcast enters `queued`, fanout uses the broadcast-owned content snapshot and does not depend on a reusable template.
