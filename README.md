# todoe

Modular monolith with Ports & Adapters architecture, append-only persistence, and event-driven side effects.

## Prerequisites

Install these first:

- Go `1.25+`
- Docker + Docker Compose
- Bun `1.x` (for frontend apps)
- `curl` (for quick API checks)

## Repo Setup

```bash
# from repo root
go mod download
go mod verify
```

Frontend dependencies:

```bash
cd web/vue && bun install
cd ../onboarding && bun install
cd ../..
```

## Environment Variables

The backend binaries use environment variables, but all of them have local defaults.

Create a `.env` (optional but recommended) in repo root:

```env
# API
MONGO_URI=mongodb://root:root@localhost:27017
NATS_URL=nats://localhost:4222

# Audit service
LOKI_URL=http://localhost:3100
```

If omitted:
- `MONGO_URI` defaults to `mongodb://root:root@localhost:27017`
- `NATS_URL` defaults to `nats://127.0.0.1:4222`
- `LOKI_URL` defaults to `http://localhost:3100`

## Start Infrastructure

Run MongoDB, NATS, Loki, Grafana:

```bash
docker compose -f compose.yml up -d
```

Exposed ports:
- MongoDB: `27017`
- NATS: `4222`
- Loki: `3100`
- Grafana: `3001` (container `3000`)

## Run Services (4 terminals)

To start the Docker infrastructure and all four backend services in one terminal:

```bash
go run ./cmd/start
```

The script starts the subscriber workers before the API and stops the Go
services when you press Ctrl+C. Docker services remain running.

Or start each backend service manually:

### 1) API

```bash
go run ./cmd/api
```

API listens on `http://localhost:3000`.

### 2) Welcome worker

```bash
go run ./cmd/welcome
```

### 3) Credit worker

```bash
go run ./cmd/credit
```

### 4) Audit worker

```bash
go run ./cmd/audit
```

## Run Frontends (optional)

### Task UI

```bash
cd web/vue
bun run dev
```

### Onboarding UI

```bash
cd web/onboarding
bun run dev
```

## Quick Verification

Health check:

```bash
curl http://localhost:3000/health
```

Create a task:

```bash
curl -X POST http://localhost:3000/tasks \
  -H "Content-Type: application/json" \
  -d '{"title":"first task","description":"setup complete"}'
```

List tasks:

```bash
curl http://localhost:3000/tasks
```

## Build & Test

```bash
go build ./cmd/api
go test ./...
```

## Observability

- Grafana: [http://localhost:3001](http://localhost:3001)
- Loki datasource is provisioned from `provisioning/`.
- Audit events are pushed by `cmd/audit` into Loki.

## Stop Everything

```bash
docker compose -f compose.yml down
```

To also remove persisted volumes:

```bash
docker compose -f compose.yml down -v
```
