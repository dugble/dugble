# Frontend API Contract Status

This document records the current dashboard API decisions that affect frontend implementation. It is intentionally explicit about contracts that are supported today versus follow-up work that is not yet available.

## Campaign scheduling

The current campaign API supports **one-time scheduling only**.

Campaigns expose a single `scheduled_at` timestamp. The current contract does not define recurrence rules, recurrence state, next-run semantics, or recurring-schedule management endpoints.

Frontend campaign builders should therefore:

- support immediate sends and one-time scheduled sends;
- use `scheduled_at` for scheduled campaigns;
- not expose daily/weekly/monthly recurrence controls yet;
- not infer recurring behavior from the existing campaign API.

Recurring campaign scheduling is a follow-up contract and must be added and verified before the frontend depends on it.

## Templates

Template categories are supported by the backend and should be treated as part of the public template contract.

Valid categories are:

- `otp`
- `welcome`
- `receipt`
- `alert`
- `notification`
- `custom`

The category is available through template create/update/list/resource flows and is preserved when templates are duplicated.

### Template usage

`sent_last_30d` is an API aggregate rather than a persisted database column. The underlying email records are linked to their originating template through `email_messages.template_id`, allowing the API to calculate recent usage.

Frontend code should treat `sent_last_30d` as optional until it is present in the documented response schema; it should not expect a database field with that name.

## Broadcast/campaign content

The current campaign contract accepts inline campaign content through `body`. It does not currently require a template reference.

The frontend should therefore not assume that every campaign must have a `template_id`. A template-backed composer can be added as a UX layer, but making templates mandatory would require an explicit API contract change.

Recommended composer behavior for the current contract:

1. Allow custom/inline campaign content.
2. Optionally allow selecting a saved template and using its rendered content.
3. Do not send a `template_id` unless the API contract explicitly adds and documents that field.

## Sender IDs

The current public sender-ID response does not expose a usage-count field. A usage metric should not be inferred from the existing sender-ID object.

If a usage count is added later, the API should document its definition and time window explicitly (for example, messages sent in the last 30 days) before the frontend displays it as a quota/usage metric.

## Pagination

The dashboard API is not globally uniform in pagination style.

- Emails, SMS, segments, and segment contacts use `limit`/`offset` pagination.
- Templates and Topics use cursor-style pagination where documented (`after`/`before`/`limit`).
- Sender Domains currently return a plain array and expose no pagination parameters.

Frontend clients should follow the contract of each endpoint rather than applying one global pagination adapter to every resource.

If sender domains later need pagination, that should be introduced as an explicit API change rather than inferred by the frontend.
