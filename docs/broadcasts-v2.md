# Broadcasts v2

This document defines the target model for Dugble email broadcasts.

The current implementation stores inline broadcast content by creating an internal message template and then making the broadcast point at that template. That couples a one-off marketing campaign to the reusable-template lifecycle and makes broadcast create, update, preview, send, and fanout depend on template-specific behavior.

Broadcasts v2 removes that coupling.

## Design principles

1. A broadcast is a complete, sendable marketing email.
2. A message template is reusable content and is optional when creating a broadcast.
3. Selecting a template copies a snapshot into the broadcast; sending does not dereference a mutable template.
4. `name` is an internal label. The recipient-facing subject is `subject`.
5. The API exposes broadcast content directly, regardless of how it is stored internally.
6. Content becomes immutable once delivery has started.

## Resource model

A broadcast owns:

- `name` — internal label
- `segment_id` — target audience
- `topic_id` — optional subscription topic
- `from_email`
- `from_name`
- `reply_to_email`
- `subject`
- `preview_text`
- `html`
- `text`
- `variable_bindings`
- lifecycle timestamps and delivery counters

A broadcast may also retain optional source provenance:

- `source_template_id`
- `source_template_version_id`

These source fields are informational. They are not the content source at send time.

## Create API

Inline content is the default path:

```json
{
  "segment_id": "segment-id",
  "name": "August product update",
  "from_email": "updates@example.com",
  "from_name": "Dugble",
  "subject": "Everything we shipped in August",
  "html": "<h1>What's new</h1>",
  "text": "What's new",
  "topic_id": "topic-id"
}
```

`name` is optional at the API boundary for inline broadcasts. If omitted or blank, the server defaults it to `subject`.

A reusable template can be used as a source, but its selected version must be copied into broadcast-owned content before the broadcast is persisted as sendable:

```json
{
  "segment_id": "segment-id",
  "name": "August product update",
  "template": "product-update"
}
```

After creation, both forms produce the same broadcast resource and follow the same send path.

## Update API

Draft and scheduled broadcasts can update their audience and content directly. Updating a scheduled broadcast must preserve the scheduled send unless it is explicitly canceled or rescheduled.

After a broadcast enters `queued`, message content is immutable. A sent broadcast may still allow its internal `name` to be changed.

## Lifecycle

```text
draft
  | send now
  v
queued -------- cancel --------> canceled
  |
  | complete
  v
sent


draft -- schedule --> scheduled
                       | cancel schedule
                       v
                      draft
                       
                       | due
                       v
                     queued
```

Target statuses:

- `draft`
- `scheduled`
- `queued`
- `sent`
- `failed`
- `canceled`

Cancel semantics:

- canceling `scheduled` removes the schedule and returns the broadcast to `draft`;
- canceling `queued` stops remaining deliveries and moves the broadcast to `canceled`;
- already delivered emails cannot be recalled.

## Storage direction

The public resource stays flat even if storage is normalized.

Preferred storage is a one-to-one content row so list queries do not fetch large HTML bodies:

```text
broadcasts
  id
  team_id
  name
  status
  segment_id
  topic_id
  source_template_id nullable
  source_template_version_id nullable
  scheduled_at
  queued_at
  sent_at
  canceled_at
  counters...
  revision
  created_at
  updated_at

broadcast_contents
  broadcast_id primary key
  team_id
  from_email
  from_name
  reply_to_email
  subject
  preview_text
  html_body
  text_body
  variable_bindings
  created_at
  updated_at
```

If implementation complexity favors keeping the content columns directly on `broadcasts`, the API and domain rules above remain unchanged.

## Send snapshot invariant

Before a broadcast can transition from `draft` or `scheduled` to `queued`, it must have a complete content snapshot:

- non-empty `subject`;
- non-empty `html` (or another supported body representation in the future);
- valid sender information;
- a valid segment;
- any required topic/compliance checks.

The worker must fan out from that broadcast-owned snapshot. It must not fetch the current version of a reusable template during delivery.

## Template migration

The existing `CreateBroadcastContent`, `BroadcastPublishedVersion`, hidden broadcast-template alias, and cleanup path in the message-template module are transitional implementation details and should be removed once all inline broadcasts use broadcast-owned content.

Existing template-based broadcasts should be migrated by copying the pinned/published template version into the broadcast content snapshot and retaining the template IDs only as source provenance where useful.

## Implementation sequence

1. Add broadcast-owned content storage and backfill existing broadcasts from their pinned/published template versions.
2. Update SQL queries and repository models to read/write broadcast content.
3. Make inline `POST /broadcasts` persist content directly; default blank `name` to `subject`.
4. Change template-based creation to snapshot the selected template version.
5. Change preview to render from broadcast content.
6. Change send/fanout to use the broadcast snapshot rather than `template_id`/`template_version_id`.
7. Remove hidden broadcast-template creation and cleanup behavior from `messagetemplate`.
8. Align scheduled edit/cancel semantics with the lifecycle above.
9. Update the public broadcast documentation and frontend contract.

## Compatibility strategy

During the transition, keep the existing `template` create/update input as an optional source to avoid breaking clients. The runtime dependency on templates should disappear before the compatibility field is considered for removal.
