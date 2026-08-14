# Dugble

Dugble is a developer-first communication platform that provides reliable API infrastructure for sending transactional email, A2P SMS, and One-Time Passwords (OTPs).

Dugble is designed to give applications a consistent communication layer while keeping delivery workflows and provider integrations behind a clean API.

## What Dugble provides

- **Transactional email** for account, product, and system events.
- **A2P SMS** for application-to-person messaging.
- **One-Time Passwords (OTPs)** for authentication and verification flows.
- **API-first infrastructure** for integrating communication into web and backend applications.

## Repository layout

- [`server/`](server/) — Go API, workers, database migrations, and backend application code.
- [`deploy/`](deploy/) — Docker Compose, Caddy, and NATS deployment configuration.
- [`docs/`](docs/) — customer dashboard API integration documentation.
- [`.github/`](.github/) — CI, security, and dependency automation.

## Getting started

### Requirements

- Docker with Docker Compose
- `make`

For local backend development, the server currently targets Go 1.26.6. Node.js 24 is pinned in [`.nvmrc`](.nvmrc) for JavaScript tooling.

### Run with Docker

Create your deployment environment file from the example and replace placeholder credentials or secrets as needed:

```sh
cp .env.example .env
make up
```

Check the running services or follow their logs:

```sh
make ps
make logs
```

Stop the stack with:

```sh
make down
```

### Run the server locally

Create the server development environment file, install dependencies, and start the API:

```sh
cp server/.env.example server/.env
cd server
go mod download
go run ./cmd/server
```

## Development commands

The root [`Makefile`](Makefile) provides the common deployment commands:

```text
make up       Build and start the stack
make down     Stop the stack
make deploy   Pull and deploy the latest main branch
make logs     Follow logs
make ps       Show service status
make migrate  Run database migrations
make restart  Restart application services
```

## Documentation

See the [Dugble API integration guide](https://dugble.com/docs/overview) for customer-facing HTTP contracts, authentication requirements, request payloads, and response formats.
