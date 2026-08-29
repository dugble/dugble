# Customer dashboard API integration guide

This directory documents the HTTP contracts used by the **customer and team
dashboard**.

It does **not** document the administrative backoffice application or its
`/backoffice` routes. Dashboard code must only use the public routes documented
here. Every request and response example in this guide must be verified against
the Go route registration, request type, handler, and service response before it
is published.

## Base URL

Use the API origin configured for the environment.

```env
NEXT_PUBLIC_API_URL=http://localhost:8080
```

Production example:

```env
NEXT_PUBLIC_API_URL=https://api.example.com
```

Paths in this guide are relative to that origin.

## Browser session requirements

The customer dashboard uses an HTTP-only session cookie. Browser requests that need the session must include credentials.

```http
Accept: application/json
Content-Type: application/json
```

For browser fetch requests, use `credentials: "include"`. The session cookie is
HTTP-only. The CSRF cookie is managed by the middleware, while the token that
frontend code sends is returned in the JSON body from `GET /csrf`.

State-changing browser requests must first obtain a CSRF token from `GET /csrf`
and send it in the `X-CSRF-Token` header. Do not persist either token in
`localStorage` or expose the session cookie to JavaScript.

## Team-scoped requests

Most dashboard resources belong to the currently selected team. Send the team
UUID in `X-Team-ID` for routes that do not already contain `:team_id` in their
path. Routes under `/teams/:team_id` derive the team from the path instead.

The API authorizes the selected team on every request. Changing `X-Team-ID` is
not an authorization mechanism, and a frontend must handle `403 Forbidden` when
membership or permissions have changed.

## Standard success envelope

Unless an endpoint returns `204 No Content`, successful JSON responses use this envelope:

```json
{
  "success": true,
  "data": {}
}
```

## Standard error envelope

```json
{
  "success": false,
  "error": {
    "code": "BAD_REQUEST",
    "message": "Public error message"
  }
}
```

Common error codes include:

- `BAD_REQUEST`
- `UNAUTHORIZED`
- `FORBIDDEN`
- `STEP_UP_REQUIRED`
- `NOT_FOUND`
- `CONFLICT`
- `PAYMENT_REQUIRED`
- `PAYLOAD_TOO_LARGE`
- `UNPROCESSABLE_ENTITY`
- `TOO_MANY_REQUESTS`
- `INTERNAL_ERROR`
- `SERVICE_UNAVAILABLE`

## Endpoint documentation format

Every endpoint entry must include:

1. HTTP method and path
2. Authentication requirement
3. CSRF requirement
4. Path parameters
5. Query parameters
6. Request payload
7. Success status and response body
8. Known error statuses and response bodies
9. Relevant behavioural notes

Do not infer fields from database tables. Payloads and responses must come from the public HTTP request and response types.

## Sections

- [Authentication](authentication.md)
- [Users](users.md)
- [Teams](teams.md)
- [Team Tokens](team-tokens.md)
- [Contacts](contacts.md)
- [Segments](segments.md)
- [Messaging APIs](messaging.md)
- [Campaigns](campaigns.md)
- [Suppressions](suppressions.md)
- [Webhooks](webhooks.md)
- [Plans](plans.md)
- [Subscriptions](subscriptions.md)
- [Wallet](wallet.md)
- [Permissions and Roles](permissions.md)

## Documentation status

The directory structure and shared contracts are established. Endpoint payloads and responses are being extracted feature by feature from the backend source. An endpoint must not be marked complete until its request and response examples have been checked against the implementation.
