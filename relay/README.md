# Dugble Relay

An open-source communications abstraction for Go.

Relay gives applications one small interface for sending communications across multiple providers without coupling business logic to provider-specific APIs, response formats, or failure semantics.

> **Safety invariant:** Relay only falls back to another provider after the previous provider definitively rejected the message. If acceptance is ambiguous, Relay stops rather than risk sending a duplicate.

## Status

Relay is experimental and the public API may change before v1.0. SMS routing is implemented with real provider adapters, and email now has production adapters for Resend and Postmark.

## Install

```bash
go get github.com/dugble/relay
```

## Core contracts

Channel-neutral delivery primitives live in the root `relay` package. Channel packages own their message models, provider request contracts, capabilities, routing, and execution.

```go
type Provider interface {
    Name() string
}

type SubmissionState string

const (
    SubmissionAccepted SubmissionState = "accepted"
    SubmissionRejected SubmissionState = "rejected"
    SubmissionUnknown  SubmissionState = "unknown"
)

type Result struct {
    Provider          string
    ProviderMessageID string
    State             SubmissionState
}
```

Common delivery errors, provider health, and observability contracts also live at the root. Channel packages alias the shared result and submission-state types where useful.

## SMS

Providers implement `sms.Provider`:

```go
type Provider interface {
    relay.Provider
    Send(context.Context, Message) (SendResult, error)
}
```

Then compose them in priority order:

```go
router, err := sms.NewRelay(primary, secondary)
if err != nil {
    log.Fatal(err)
}

result, err := router.Send(ctx, sms.Message{
    To:      "+233200000000",
    From:    "Acme",
    Text:    "Your verification code is 492031",
    Purpose: sms.PurposeVerification,
})
```

## Email

Email uses the same root result, health, observability, and fallback semantics as SMS while keeping an email-specific message model and capability contract.

```go
router, err := email.NewRelay(primary, secondary)
if err != nil {
    log.Fatal(err)
}

result, err := router.Send(ctx, email.Message{
    From: email.Address{Email: "noreply@example.com", Name: "Acme"},
    To: []email.Address{
        {Email: "customer@example.com"},
    },
    ReplyTo: &email.Address{Email: "support@example.com"},
    Subject: "Your receipt",
    Text:    "Thanks for your order.",
    HTML:    "<p>Thanks for your order.</p>",
})
```

Email V1 intentionally covers transactional fields only: From, To, Reply-To, Subject, text body, and HTML body. Attachments, CC/BCC, templates, tracking settings, tags, custom headers, scheduling, and marketing concepts are not part of the core contract yet.

Providers may optionally expose `email.Capabilities` for HTML, Reply-To, multiple-recipient support, provider-specific subject requirements, and recipient-count limits. Unsupported providers are skipped before a network request.

## Submission states

Every provider must normalize submission into one of three states:

| State | Meaning | Relay behavior |
| --- | --- | --- |
| `accepted` | The provider definitely accepted the message. | Stop. |
| `rejected` | The provider definitely did not accept the message. | A fallback may be attempted. |
| `unknown` | The provider may have accepted the message. | Stop; never fallback. |

An empty or unrecognized state is treated as `unknown` conservatively.

## Provider capabilities

Channel packages can expose capability contracts that Relay evaluates before making a network request. Providers without capability metadata remain eligible for backward compatibility and simple adapters.

SMS capabilities cover alphanumeric sender IDs, sender-ID length, and E.164 recipient requirements. Email capabilities cover HTML, Reply-To, multiple recipients, required subject lines, and maximum recipient counts.

## Provider health

Health is a routing input, not provider configuration or a delivery result. Relay exposes the channel-agnostic `relay.HealthSource` contract with three states:

- `healthy` — preferred for new traffic
- `degraded` — still eligible, but tried after healthy providers
- `unavailable` — excluded from the route

```go
router = router.WithHealth(relay.HealthFunc(func(ctx context.Context, provider string) relay.HealthStatus {
    return currentHealth(provider)
}))
```

Unknown health values are treated as healthy for backward compatibility. A provider being disabled is configuration state and intentionally does not belong in health.

## Observability

Relay exposes dependency-free, channel-neutral lifecycle events through `relay.Observer`:

```go
router = router.WithObserver(relay.ObserverFunc(func(ctx context.Context, event relay.Event) {
    record(event)
}))
```

SMS and email emit `route_selected`, `provider_skipped`, `attempt_started`, `attempt_finished`, and `route_exhausted`. Attempt-finished events include the normalized outcome, provider message ID, duration, and a `HadError` flag. Skip and exhaustion events use typed machine-readable reasons. Events identify their channel as `sms` or `email`.

Observer callbacks run synchronously and should return quickly; move expensive logging or metrics export to your own queue or worker. Observer panics are isolated from delivery. Events intentionally omit message bodies, recipient/sender values, and raw provider errors so provider-response payloads are not handed to telemetry by default.

## Hubtel

```go
provider, err := hubtel.New(hubtel.Config{
    ClientID:     os.Getenv("HUBTEL_CLIENT_ID"),
    ClientSecret: os.Getenv("HUBTEL_CLIENT_SECRET"),
})
```

Hubtel's adapter maps a documented `201 Created` response to `accepted`. Documented client-side failures (`400`, `401`, `404`, and `429`) are treated as definite rejection. Transport failures, server errors, and unexpected HTTP statuses are treated as `unknown`, so Relay will not fail over and risk a duplicate SMS.

## mNotify

```go
provider, err := mnotify.New(mnotify.Config{
    APIKey: os.Getenv("MNOTIFY_API_KEY"),
})
```

The adapter uses mNotify's quick-send endpoint and sends `recipient`, `sender`, `message`, `is_schedule`, and `schedule_date`. A `200 OK` response with `status: "success"` and code `2000` is normalized to `accepted`.

mNotify's public quick-send material clearly documents the success shape, but does not provide sufficient acceptance guarantees for every failure response. Relay therefore treats transport failures, malformed responses, and non-success API responses as `unknown` rather than automatically failing over and risking duplicate delivery.

## Resend

```go
provider, err := resend.New(resend.Config{
    APIKey: os.Getenv("RESEND_API_KEY"),
})
```

Resend's adapter uses `POST /emails` and declares provider-specific capabilities: HTML, Reply-To, multiple recipients, a required subject, and at most 50 `To` recipients.

A documented `200 OK` is normalized to `accepted` even when the response body cannot be decoded, because falling back after Resend has accepted the email could duplicate delivery. Explicit documented validation, auth, quota, and rate-limit failures are normalized to `rejected`. Transport failures, `408`, server errors, unexpected statuses, and `409 concurrent_idempotent_requests` remain `unknown`.

## Postmark

```go
provider, err := postmark.New(postmark.Config{
    ServerToken: os.Getenv("POSTMARK_SERVER_TOKEN"),
})
```

Postmark's adapter uses the single-email `POST /email` endpoint with server-token authentication. The message stream is adapter configuration and defaults to Postmark's `outbound` transactional stream. The adapter supports HTML, Reply-To, and up to 50 `To` recipients; Postmark does not require a subject.

A documented `200 OK` is normalized to `accepted` even if the response body cannot be decoded. Documented request-side failures (`401`, `404`, `413`, `415`, `422`, and `429`) are normalized to `rejected`. Transport failures, `408`, server errors (`500` and `503`), and undocumented statuses remain `unknown`, so Relay will not fall back when acceptance may be ambiguous.

For production calls, provide a context with an appropriate deadline.

## Why `unknown` matters

A network timeout does not prove that a message was not submitted. The provider may have accepted the message and the response may have been lost. Retrying through another provider in that situation can produce duplicate OTPs, transactional SMS, or email.

Relay makes this ambiguity explicit instead of reducing every provider call to `error != nil`.

## Mock provider

`providers/mock` is included for SMS application and adapter tests:

```go
provider := mock.New("primary", func(ctx context.Context, message sms.Message) (sms.SendResult, error) {
    return sms.SendResult{State: sms.SubmissionAccepted}, nil
})
```

## Provider contract tests

`relaytest` provides a reusable, transport-agnostic contract for production provider adapters. Each adapter supplies fresh accepted and unknown fixtures, plus an optional rejected fixture when the provider exposes a failure that is documented strongly enough to permit fallback.

The contract verifies both normalized provider states and their channel Relay consequences:

```text
accepted -> fallback is not called
rejected -> fallback is called
unknown  -> fallback is not called
```

Provider-specific HTTP fixtures stay in the adapter's own tests. Rejected coverage is intentionally optional so adapters are never encouraged to classify an ambiguous provider failure as safe merely to satisfy the test harness.

## Initial roadmap

- [x] SMS message model
- [x] Provider contract
- [x] Normalized submission states
- [x] Ordered routing
- [x] Safe fallback semantics
- [x] Mock provider
- [x] Routing tests
- [x] Provider capabilities
- [x] Provider health
- [x] Hubtel adapter
- [x] mNotify adapter
- [x] Twilio adapter
- [x] Vonage adapter
- [x] Observability hooks
- [x] Root package cleanup
- [x] Email contracts
- [x] First email provider adapter (Resend)
- [x] Provider contract test harness
- [x] Second email provider adapter (Postmark)

## Package boundary

```text
relay/
├── relay.go
├── errors.go
├── result.go
├── health.go
├── observability.go
│
├── sms/
│   ├── message.go
│   ├── provider.go
│   ├── capabilities.go
│   ├── routing.go
│   └── relay.go
│
└── email/
    ├── message.go
    ├── provider.go
    ├── capabilities.go
    ├── routing.go
    └── relay.go
```

Tests and provider adapters are omitted from the diagram for clarity.

## Design boundary

Relay is intended to stay focused on provider abstraction and delivery decisions. Persistence, billing, accounts, sender-ID onboarding, domains, webhooks, and managed compliance belong in products built on top of Relay rather than in the core library.

## License

Apache-2.0.
