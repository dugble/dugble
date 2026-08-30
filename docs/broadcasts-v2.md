# Broadcasts v2

This document records the implemented Dugble email broadcast architecture.

Broadcasts v2 removes the old runtime coupling between one-off marketing broadcasts and reusable message templates. A Broadcast is now the sendable message resource: it owns its audience definition, sender fields, subject, preview text, HTML/text bodies, variable bindings, lifecycle state, and delivery counters.

## Design principles

1. A broadcast is a complete, sendable marketing email.
2. Reusable message templates are separate resources.
3. A client may copy reusable template content into a broadcast before creation, but the Broadcast API does not persist template references.
4. Sending, preview, recipient fanout, retries, and analytics operate on broadcast-owned content.
5. `name` is an internal label; `subject` is recipient-facing.
6. Broadcast message content is editable only while status is `draft` or `scheduled`.
7. Once a broadcast reaches `queued`, delivery uses the immutable owned-content snapshot and never dereferences a mutable template.

## Resource model

A Broadcast owns:

- `name`
- `status`
- `segment_id`
- `topic_id`
- `from_email`
- `from_name`
- `reply_to_email`
- `subject`
- `preview_text`
- `html`
- `text`
- `variable_bindings`
- `scheduled_at`
- `queued_at`
- `sent_at`
- `canceled_at`
- audience / eligible / suppressed / queued / failed counters
- `revision`
- created / updated timestamps

There are no `template_id`, `template_version_id`, `source_template_id`, or `source_template_version_id` fields on the Broadcast resource or Broadcast persistence model.

## Storage

Broadcast-owned message content is stored directly on `broadcasts`:

```text
broadcasts
  id
  team_id
  name
  status
  segment_id
  topic_id
  from_email
  from_name
  reply_to_email
  subject
  preview_text
  html_body
  text_body
  variable_bindings
  scheduled_at
  queued_at
  sent_at
  canceled_at
  recipients_materialized_at
  audience_count
  eligible_count
  suppressed_count
  queued_count
  failed_count
  revision
  created_at
  updated_at
  deleted_at
```

Recipient fanout claims the broadcast message fields together with each materialized recipient. The delivery worker therefore renders and queues email from the claimed broadcast snapshot without consulting `message_templates` or `message_template_versions`.

## Create

`POST /broadcasts` accepts owned message content directly.

```json
{
  "segment_id": "segment-id",
  "name": "August product update",
  "from_email": "updates@example.com",
  "from_name": "Dugble",
  "reply_to_email": "support@example.com",
  "subject": "Everything we shipped in August",
  "preview_text": "A quick recap",
  "html": "<h1>Hello {{{FIRST_NAME}}}</h1>",
  "text": "Hello {{{FIRST_NAME}}}",
  "variable_bindings": {
    "FIRST_NAME": "there"
  },
  "topic_id": "topic-id"
}
```

`name` is optional and defaults to `subject`. `segment_id`, `subject`, and `html` are required by the service. The same create request can queue immediately with `send: true`, or schedule with `send: true` plus future `scheduled_at`.

Reusable templates are intentionally not accepted as Broadcast API references. A caller that wants to use a template must resolve/copy its desired content before creating or updating the broadcast.

## Update

`PATCH /broadcasts/:broadcast` edits audience and owned content directly. The request requires the current positive `revision` for optimistic concurrency.

Both `draft` and `scheduled` broadcasts are editable. Updating a scheduled broadcast preserves the existing schedule. Nullable fields distinguish omission from explicit `null`, so clients can either leave a field unchanged or clear it.

`queued`, `sent`, `failed`, and `canceled` broadcasts are not editable through the update path.

## Preview and rendering

`POST /broadcasts/:broadcast/preview` renders directly from the Broadcast resource.

Broadcast `variable_bindings` provide base values and supplied preview/fanout variables override them. Current placeholder syntax is triple-brace identifier keys such as:

```text
{{{FIRST_NAME}}}
```

HTML substitutions are escaped. Subject, preview-text, and plain-text substitutions are not HTML-escaped. Missing referenced variables are render errors.

The broadcast renderer intentionally performs no message-template lookup.

## Lifecycle

```text
draft
  | send now
  v
queued -------- cancel --------> canceled
  |
  | complete with no failures
  v
sent
  |
  +-- terminal fanout failures produce failed instead


draft -- schedule --> scheduled
                       | cancel schedule
                       v
                      draft

scheduled -- due ----------------> queued
scheduled -- reschedule ---------> scheduled
```

Statuses:

- `draft`
- `scheduled`
- `queued`
- `sent`
- `failed`
- `canceled`

Cancel semantics:

- canceling `scheduled` clears the schedule and returns the broadcast to `draft`;
- canceling `queued` moves the broadcast to `canceled` and prevents remaining fanout work;
- email already queued into the email delivery subsystem cannot be recalled.

Delete semantics:

- only `draft` and `canceled` broadcasts can be soft-deleted.

## Recipient materialization and fanout

When a broadcast is queued, recipient materialization snapshots the target audience into `broadcast_recipients`, including exclusions. Materialization records audience, eligible, and suppressed counts on the Broadcast.

Fanout then claims pending recipients together with the exact broadcast-owned sender/content fields. For each recipient it:

1. overlays recipient-specific variables onto `variable_bindings`;
2. renders the owned subject/body fields;
3. enqueues an email message;
4. records the resulting email message ID or a terminal/retryable failure;
5. finalizes the Broadcast to `sent` or `failed` when no pending recipients remain.

This path is independent of reusable templates.

## Message-template boundary

The old hidden broadcast-template compatibility layer has been removed from `messagetemplate`:

- no `CreateBroadcastContent` helper;
- no `BroadcastPublishedVersion` helper;
- no broadcast-specific cleanup helper;
- no `__broadcast_` alias convention;
- no special template-list filtering for hidden broadcast content;
- no broadcast-specific SQL cleanup query.

`messagetemplate` now serves reusable templates only.

## API surface

The Broadcast routes are:

- `POST /broadcasts`
- `GET /broadcasts`
- `GET /broadcasts/:broadcast`
- `PATCH /broadcasts/:broadcast`
- `DELETE /broadcasts/:broadcast`
- `POST /broadcasts/:broadcast/send`
- `POST /broadcasts/:broadcast/cancel`
- `POST /broadcasts/:broadcast/duplicate`
- `POST /broadcasts/:broadcast/preview`
- `GET /broadcasts/:broadcast/recipients`
- `GET /broadcasts/:broadcast/exclusions`
- `GET /broadcasts/:broadcast/analytics`

The detailed request/response contract is documented in `docs/broadcasts.md`.

## Remaining compliance integration

Managed unsubscribe link generation is a separate delivery/compliance integration. The broadcast content model and fanout architecture support managed variables, but this document does not claim a production unsubscribe linker exists until that endpoint/linker is implemented and wired.
