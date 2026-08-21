# CLAUDE.md — todoe

Modular monolith, Ports & Adapters (hexagonal) architecture, append-only event store. See `principles.md` for full rationale.

---

## Folder Structure

Each domain follows this layout exactly:

```
infra/<domain>/
├── domain/          — models, constants, value objects
├── port/            — UseCase (PortIn) and Repository (PortOut) interfaces
├── adapter/
│   ├── *.go         — infrastructure adapters (e.g. mongo_repository.go)
│   └── http/        — HTTP transport, package httpadapter
└── application/     — Service struct (orchestration + pure logic helpers)

cmd/api/             — binary entry point; wires all adapters
infra/               — one subdirectory per domain, no cross-domain imports
```

---

## Three Function Types

Every function must be one of these three types — never mixed.

| Type | Rule | Where |
|---|---|---|
| **Pure Logic** | No I/O, no side effects. Returns plain values. | `application/`, `domain/` |
| **Side Effect** | Touches DB, network, filesystem, time, random. MUST return `mo.Result[T]` or `mo.Option[T]`. **NEVER** `(T, error)`. | `adapter/` |
| **Orchestration** | Composes Pure + Side Effect. Returns `mo.Result[T]`. Named after use cases. | `application/` (Service methods) |

- A function calling `time.Now()` is a Side Effect.
- Pure validation (no I/O) is Pure Logic even if it produces an error value.
- Never put business logic inside a Side Effect function.

---

## Domain Events — Publish-Only Pattern

```
Service.DoThing()
  ├── validate input           ← Pure Logic
  ├── build domain entity      ← Pure Logic (pre-generate ID: primitive.NewObjectID())
  ├── publisher.Publish(event) ← Side Effect — the only boundary crossing
  └── return entity            ← caller gets complete result immediately

EventBus dispatches to:
  ├── SaveHandler          → repo.Create(entity)
  ├── NotificationHandler  → downstream side effect
  └── LogHandler           → wildcard
```

**Rules:**
- Service NEVER calls `repo.Create` directly.
- Service NEVER calls another Service directly.
- Adding a side effect = new handler + `bus.Subscribe`. Publishing service does not change.
- Pre-generate IDs before `Publish` so the caller receives a complete entity.
- Service test: assert event published — no repo mock needed.
- Handler test: assert `repo.Create` called — no service involved.

---

## Repository Rules

- Append-only. No `Update`, no `Delete` methods on any Repository interface.
- An "update" creates a new record referencing the original ID.
- A "delete" creates a tombstone record.
- Each repository owns exactly one MongoDB collection.
- Interface lives in `port/`. MongoDB implementation lives in `adapter/`.

---

## Port / Adapter Rules

- `port/` contains interfaces only — no structs, no framework imports.
- UseCase methods named after business actions: `RegisterCustomer`, `PlaceOrder` — not `Create`, `Get`, `Update`.
- Compile-time check in `application/service.go`: `var _ port.UseCase = (*Service)(nil)`
- HTTP handlers in `adapter/http/` depend on `port.UseCase`, never on `*application.Service`.
- Validation errors (`ErrInvalidX`) defined in `application/`; HTTP status mapping in `adapter/http/`.
- Pure infrastructure helpers (`parseObjectID`, etc.) live in `application/` as Pure Logic.

---

## Key Decisions

- `github.com/samber/mo` — only mechanism for error/optional returns from Side Effect functions.
- No shared services across domains.
- MongoDB is the backing store; `primitive.NewObjectID()` for all entity IDs.
- `package httpadapter` is the canonical package name for `adapter/http/*.go`.
- No `vendor/` directory — use the module cache.

---

## Commands

```bash
go build ./cmd/api        # build binary
go test ./...             # run all tests
go get <pkg>@latest && go mod tidy   # add a dependency
go mod verify             # verify module integrity
```
