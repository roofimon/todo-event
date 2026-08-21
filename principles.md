# Architecture

## Architecture Patterns
- USE Ports and Adapters pattern
- Port - PortIn - PortOut - Interface
- Adapter - AdapterIn - AdapterOut - Mostly Side Effect
- Application - Orchrestration, Pure Logic Function
- Domain - Domain Model

## Architecture Rule
- Port defines contracts, not frameworks
- Application implements use case contracts
- Adapters implement transport and infrastructure details
- HTTP handler functions should not be defined as port interfaces
- Handler dependencies should be use case interfaces, not concrete service structs

## Folder Structure
```
domain/<name>/
├── domain/          — Domain models and constants
├── port/            — PortIn (UseCase) and PortOut (Repository) interfaces
├── adapter/
│   ├── *.go         — Infrastructure adapters (e.g. mongo_repository.go)
│   └── http/        — Transport adapter (HTTP handlers), package httpadapter
└── application/     — Service (Orchestration + Pure Logic)

pkg/                 — Shared infrastructure utilities (not domain services)
├── event/           — Synchronous in-process event bus
└── ...

cmd/api/             — Binary entry point; wires all domains and adapters
infra/               — Project infrastructure packages
```

## Design Decisions
- `port/` holds both PortIn (`UseCase`) and PortOut (`Repository`) in one file
- UseCase methods are named after use cases, not CRUD — e.g. `ViewCustomerDetail` not `GetByID`
- `var _ port.UseCase = (*Service)(nil)` compile-time check that Service satisfies UseCase
- HTTP handlers live in `adapter/http/` as `package httpadapter`; depend on `port.UseCase`, never on concrete `*Service`
- `parseObjectID` and similar infrastructure helpers live in `application/` as Pure Logic (no I/O)
- Validation errors (`ErrInvalidName`, `ErrInvalidEmail`, `ErrInvalidObjectID`) defined in `application/`; error-to-HTTP mapping in `adapter/http/`

## Coding Style
- Function must do one thing well
- Function can be categorized into 3 types including Pure Logic, Side Effect and Orchrestration
- Side Effect Function is function that interact with outside world like terminal, file, database, cache, queue and etc
- Pure Logic Function is function that do only business logic nothing to interacwith outside world. Mostly it will sit in the middle between side effects functions.
- Orchrestration Function is function that compose both side effect and pure logic function together so it mostly focus on larger business flow.
- Side Effect function must not return multiple value like (value, err) it must be wrapped with Result, Option or Either.

## Back End
- Modular Monolith — each domain lives under `domain/<name>/`
- NO shared services between domains
- Self Contain API, Repository and Model within domain
- `samber/mo` for Result/Option/Either types
- `mo.IOEither[R]` wraps `func() (R, error)` for fallible side-effect computations (e.g. MongoDB client); `Run()` returns `Either[error, R]`
- Lazy singleton pattern: wrap `mo.IOEither` with `sync.Once` so the computation runs at most once (connect on first use, not at startup)

## Repository
- Responsible only in it own collection
- Repository using Event Sourcing Pattens there is no update and delete
- Update or Delete operation, it will create new record that associate with an origin record

## Service
- In case that we need to compose data from multiple collections it must be done in Service
- Example like Model enrichment

## Domain Events (Publish-Only Pattern)

A service does its work, fires an event, and stops.
It never calls `repo.Create` on itself or pokes another service directly.
Side effects happen in event handlers — separate, subscribable, ignorant of each other.

```
Service.DoThing()
  ├── validate input
  ├── build domain entity  (pre-generate ID with bson.NewObjectID())
  ├── publisher.Publish(event)   ← only crossing the boundary
  └── return entity              ← caller gets the result immediately

EventBus (pkg/event.Bus)
  ├── SaveHandler          → repo.Create(entity)
  ├── NotificationHandler  → downstream side effect
  └── LogHandler           → wildcard, logs every event
```

Adding a new side effect = write a new handler and call `bus.Subscribe`. The service that published the event does not change.

**Pre-generating IDs:** Because saving is deferred to a handler, generate the ID before publishing so the caller gets a complete entity back.

**Testing:** Service tests only assert the event was published — no repo mock needed. Handler tests only assert `repo.Create` was called correctly.

**Wiring (cmd/api/main.go):**
```go
bus := event.NewBus()
repo := adapter.NewMongoRepository(clientIO)
bus.Subscribe(domain.EventCreated, adapter.NewSaveHandler(repo))
service := application.NewService(repo, bus)
```

| Event | Subscribers |
|---|---|
| `task.created` | `SaveHandler`, `LogHandler` |
| `order.placed` | `OrderSaveHandler`, `NotificationEventHandler`, `LogHandler` |
| `notification.sent` | `NotificationSaveHandler`, `LogHandler` |
| `customer.registered` | `NotificationEventHandler`, `LogHandler` |

## MongoDB
- Driver: `go.mongodb.org/mongo-driver/v2`
- ObjectID type: `bson.ObjectID` (not `primitive.ObjectID` — `primitive` subpackage does not exist in v2)
- Generate new ID: `bson.NewObjectID()`
- Lazy client: `mo.NewIOEither(func() (*mongo.Client, error) { return mongo.Connect(...) })` wrapped with `sync.Once` in the repository

## Front End
- Angular 19 standalone components, signals for state
- Small components; lazy-loaded routes
- Coinbase-inspired design system (`web/src/_coinbase-tokens.scss`)
- Top sticky dark nav; pill buttons; Active/Inactive tab bars per section
