# Backend structure

Dugble follows [Leamout's](https://github.com/leamout/leamout) approach of
organizing business capabilities into bounded areas, with separate runtime,
integration, security, and infrastructure packages. The areas reflect Dugble's
email and SMS product.

## Package ownership

Paths in this table are relative to `server/internal/`.

| Area | Existing packages | Responsibility |
| --- | --- | --- |
| `identity/` | `auth`, `mfa`, `sessions`, `users` | User identity, sign-in, and sessions. |
| `tenancy/` | `teams`, `tokens` | Team ownership, membership, and API credentials. |
| `messaging/` | `email`, `sms`, `domains`, `senderids`, `templates`, `suppressions`, `topics`, `unsubscribe` | Messaging APIs, sender configuration, and recipient preferences. |
| `audience/` | `contacts`, `properties`, `segments` | Contact data and audience selection. |
| `campaigns/` | `campaigns`, `broadcasts` | Campaign definitions and broadcast orchestration. |
| `commercial/` | `charges`, `payments`, `plans`, `subscriptions`, `wallet` | Pricing, usage charges, subscriptions, and balances. |
| `delivery/` | `attempt`, `broadcast`, `email`, `sms`, `feedback`, `webhook` | Queued execution, retries, provider feedback, and delivery state. |
| `backoffice/` | Administrative resource modules | Staff-facing business operations and HTTP handlers. |
| `modules/` | `audit`, `outbox`, `webhooks` | Capabilities shared across business areas. |
| `integrations/` | `amazon`, `dns`, `hubtel`, `leamout`, `mnotify`, `monitoring`, `moolre`, `nats`, `postgres`, `redis`, `runnage`, `security` | Concrete provider and infrastructure clients. |
| `platform/` | `config`, `event`, `httpclient`, `idempotency` | Configuration and shared infrastructure primitives. |
| `runtime/` | `server`, `worker`, `backoffice`, `http`, `middleware`, `provider` | Application composition, lifecycle, routing, middleware, and inbound provider HTTP endpoints. |
| `security/` | Root cryptographic helpers, `authn`, `authz`, `keyrotation` | Credentials, access decisions, encryption, and key rotation. |
| `database/` | `queries`, `sqlc` | SQL query sources and generated database code. |

The API, worker, and backoffice entrypoints in `cmd/` call their corresponding
`runtime/` packages. `cmd/keyrotate` runs the key-rotation service.
`runtime/http` supplies the router and HTTP lifecycle shared by the API and
backoffice, so neither application needs to import the other's startup wiring.

## Package boundaries retained during the move

The structure refactor relocates existing packages and preserves their Go
package identifiers, exported APIs, and behavior. Some shared contracts remain
in subpackages because their callers include both business code and providers:

| Previous path under `internal/` | Current path | Purpose |
| --- | --- | --- |
| `modules/domainclaim` | `messaging/domains/claims` | Domain ownership claims. |
| `modules/emailtenant` | `messaging/email/tenants` | Provisioning provider-side email tenants, including SES tenant state and jobs. |
| `platform/awsses` | `messaging/email/provider` | Email and domain provider contracts, limits, and routing metadata. |
| `platform/sms` | `messaging/sms/provider` | SMS provider contracts and sending; routing remains in its `routing/` subpackage. |
| `platform/senderid` | `messaging/senderids/provider` | Sender-ID provider contracts and validation. |
| `platform/systemmail` | `messaging/email/systemmail` | Application notifications and embedded email templates. |
| `platform/unsubscribe` | `messaging/unsubscribe` | Signed unsubscribe links. |
| `platform/audit` | `modules/audit` | Audit recording and persistence. |
| `modules/auditevent` | `modules/audit/events` | Customer audit-event query API. |
| `platform/outbox` | `modules/outbox` | Durable publication and relay. |
| `platform/webhook` | `modules/webhooks/events` | Webhook event contracts, emission, and signing-secret helpers. |
| `modules/health` | `runtime/server/health` | API liveness and readiness routes. |
| `transport/health` | `runtime/http/health` | Existing shared health handler. |

Provider implementations live in `integrations/` and use the contracts owned
by messaging. For example, `integrations/amazon/ses` implements the contracts
in `messaging/email/provider`, while `delivery/email` handles queued execution.
This move preserves the existing dependency graph; further decoupling of
individual modules can happen separately.

`platform/event` retains the existing shared event catalog. Audit and webhook
query APIs remain separate from their shared recording and emission packages.
Database migrations, SQL generation paths, public routes, event subjects,
environment variables, and executable paths retain their existing contracts.

## Module file convention

Add files when the capability needs them:

| File | Responsibility |
| --- | --- |
| `model.go` | Domain models, states, errors, commands, and inputs. |
| `service.go` | Application and business logic. |
| `repository.go` | Persistence contracts and implementation. |
| `handler.go` | HTTP request and response handling. |
| `routes.go` | Route registration. |
| `validation.go` | Request and domain validation. |
| `consumer.go` | Events entering the module. |
| `publisher.go` | Events leaving the module. |
| `jobs.go` | Background jobs. |
| `*_test.go` | Tests named for the behavior they verify. |

`internal/policy_test.go` checks the vertical modules in the new business areas,
including the relocated nested domain-claim, email-tenant, audit-event, and
health modules. Infrastructure, provider contracts, and delivery processors
retain their responsibility-specific filenames.

## Validation

Run from `server/` using the Go version in `go.mod`:

```sh
go test -race ./...
go vet ./...
go build ./cmd/...
```

Database integration tests require `TEST_DATABASE_URL` pointing to a disposable
PostgreSQL database. Without it, those tests skip. CI supplies PostgreSQL and
also checks formatting, lint, module tidiness, and sqlc generation.
