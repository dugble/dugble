# Broadcasts

Customer dashboard HTTP contract for email broadcasts.

## `POST /broadcasts`

Creates a draft broadcast. A broadcast can reference an existing reusable template, or provide inline email content from the broadcast composer. Do not send both forms in the same request.

### Inline composer payload

```json
{
  "name": "August product update",
  "segment_id": "0f593c7a-167e-4fe0-aeb8-6be39078d0f0",
  "topic_id": null,
  "subject": "What's new this month",
  "preview_text": "A quick look at what shipped.",
  "from_name": "Dugble",
  "from_email": "hello@example.com",
  "html": "<p>Hello from Dugble.</p>",
  "text": "Hello from Dugble."
}
```

For inline content, `subject` and `html` are required. `text`, `preview_text`, `from_name`, `from_email`, and `topic_id` are optional. Preview text is encoded into the stored HTML as an email preheader.

The backend creates and publishes an internal versioned template for inline content. That template is an implementation detail and is excluded from normal `GET /templates` results. Creating or sending an inline broadcast is authorized with Broadcast permissions rather than separate Templates permissions.

### Existing-template payload

```json
{
  "name": "August product update",
  "segment_id": "0f593c7a-167e-4fe0-aeb8-6be39078d0f0",
  "template": "monthly-update",
  "variable_bindings": {}
}
```

The existing template flow remains supported for API clients that intentionally reuse Templates.

## `PATCH /broadcasts/:broadcast`

Draft broadcasts can be updated with the normal metadata fields. To replace inline composer content, send a complete inline content snapshot including `subject` and `html`, along with the current `revision`.

```json
{
  "revision": 2,
  "name": "August product update",
  "segment_id": "0f593c7a-167e-4fe0-aeb8-6be39078d0f0",
  "subject": "What's new this month",
  "preview_text": "Updated preview text",
  "from_name": "Dugble",
  "from_email": "hello@example.com",
  "html": "<p>Updated content.</p>",
  "text": "Updated content."
}
```

Inline fields cannot be combined with `template` in the same update. Updating inline content creates a fresh internal content snapshot and points the draft broadcast at it; reusable templates are never mutated by the composer.

## `POST /broadcasts/:broadcast/send`

Queues a draft immediately when `scheduled_at` is omitted, or schedules it when a future timestamp is supplied.

```json
{
  "scheduled_at": "2026-09-01T09:00:00Z"
}
```

Inline broadcasts use their internally published content version and do not require separate Templates read permission to send.

## Audience topics

`topic_id` remains optional. The dashboard composer may omit it and target the selected `segment_id` directly.
