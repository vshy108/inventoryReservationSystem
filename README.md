# Inventory Reservation System

A small Go service that prevents overselling during high-concurrency flash
sales. Reservations are temporary holds on inventory that can be
**confirmed** into a sale, **cancelled** back to stock, or **expired**
automatically after a 2-minute hold.

## Quickstart

```sh
go mod tidy
go test ./...
go test -race ./...
```

Run the demo:

```sh
go run ./cmd/app
```

Run the HTTP server (default `:8080`, seeds `sku-1` with stock 1, sweeps
expired reservations every 10s):

```sh
go run ./cmd/server
# custom: go run ./cmd/server --addr=:9000 --seed="sku-1:5,sku-2:2" --hold=2m
```

Endpoints:

| Method | Path                              | Description            |
|--------|-----------------------------------|------------------------|
| POST   | `/reservations`                   | Reserve one unit       |
| POST   | `/reservations/{id}/confirm`      | Confirm a reservation  |
| POST   | `/reservations/{id}/cancel`       | Cancel a reservation   |
| GET    | `/reservations/{id}`              | Fetch a reservation    |
| GET    | `/products/{id}/stock`            | Available stock        |
| GET    | `/healthz`                        | Liveness probe         |

`make` targets: `make vet`, `make test`, `make race`, `make cover`, `make cover-html`, `make demo`, `make server`.

## Test Coverage

All business logic under `internal/...` is at **100% statement coverage**
(the `cmd/` entrypoints are excluded as they are thin wiring).

```sh
make cover        # prints total coverage
make cover-html   # opens line-by-line report in a browser
```

## Architecture

```
interface  ->  application  ->  domain
                    ^
              infrastructure
```

- `internal/domain/` — pure types (`Reservation`, `ProductInventory`),
  state machine, sentinel errors.
- `internal/application/` — `ReservationService` orchestrating the
  lifecycle. Serializes read-check-write via a per-product mutex.
- `internal/infrastructure/` — in-memory repository, `LockManager`,
  `Clock` abstraction (`SystemClock`, `FakeClock`).
- `cmd/app/` — runnable demo.

## Features by Level

| Level | Feature | Status |
|-------|---------|--------|
| 1 | Basic in-memory reservation; rejects when out of stock | Done |
| 2 | Lifecycle: `Active / Confirmed / Cancelled / Expired`; 2-minute hold; lazy + explicit expiry | Done |
| 3 | Concurrency safety; 500 parallel reserves on stock=1 yields exactly 1 success | Done |

## Spec -> Test -> Code Traceability

| Spec | Rules | Tests | Implementation |
|------|-------|-------|----------------|
| [specs/01-stock-formula.md](specs/01-stock-formula.md) | SF-R1..R4 | `TestReserveItem_*`, `TestCancelReservation_ReleasesStock` | `internal/domain/inventory.go` |
| [specs/02-reservation-lifecycle.md](specs/02-reservation-lifecycle.md) | RL-R1..R6 | `TestConfirmReservation_*`, `TestCancelReservation_*`, `TestExpireReservations_*`, `TestConfirmAfterExpiry_Fails` | `internal/application/reservation_service.go` |
| [specs/03-reserve-item.md](specs/03-reserve-item.md) | RI-R1..R4 | `TestReserveItem_SucceedsWhenStockAvailable`, `TestReserveItem_FailsWhenStockUnavailable`, `TestReserveItem_UnknownProduct` | `ReservationService.ReserveItem` |
| [specs/04-confirm-cancel.md](specs/04-confirm-cancel.md) | CC-R1..R7 | `TestConfirmReservation_Finalizes`, `TestConfirmReservation_DoubleConfirm`, `TestCancelReservation_ReleasesStock` | `ConfirmReservation`, `CancelReservation` |
| [specs/05-expiry.md](specs/05-expiry.md) | EX-R1..R5 | `TestExpireReservations_ReleasesExpiredStock`, `TestConfirmAfterExpiry_Fails` | `ExpireReservations`, `expireReservationLocked` |
| [specs/06-concurrency.md](specs/06-concurrency.md) | CO-R1..R4 | `TestReserveItem_500Concurrent_Stock1` + `go test -race` | `LockManager`, `ReservationService.ReserveItem` |

## Design Decisions & Trade-offs

- **Per-product `sync.Mutex` over optimistic retry.** Simpler to audit
  for correctness in an interview context, and already scales on the
  axis that matters (per-SKU). Different products contend independently.
- **In-memory only.** The brief states "maintain inventory in memory".
  The repository is behind a narrow type so a durable store can be added
  later without touching the domain or service logic.
- **Lazy expiry inside `ReserveItem`.** Avoids needing a background
  timer goroutine to make the availability check correct. A separate
  `ExpireReservations` sweep is still provided for explicit use.
- **`Clock` abstraction.** All time reads go through an interface so
  tests stay deterministic. `FakeClock.Advance` drives expiry scenarios
  without `time.Sleep`.
- **Sentinel errors over wrapped types.** Keeps the domain surface
  minimal; callers use `errors.Is`.
- **Confirmed is final.** The brief says confirmed purchases cannot be
  reversed, so `Confirm` has no inverse.

## TDD Approach

Specs in [specs/](specs/) define the rules first; every rule has a
direct test in the traceability table above. The initial Level 1/2/3
scaffold was committed together (`816dcf1`) to keep the baseline
compilable, but each subsequent feature followed a **test-paired
commit flow**:

- `91a34b0 docs(specs): add spec files with traceability to tests` —
  specs authored before the corresponding extension features.
- `14d3346 feat(domain): add inventory event log and extended tests` —
  event log code and its tests (`TestInventoryEventLog_RecordsTransitions`,
  `TestReserveAfterExpiry_FreesSlot`, `TestCancelThenReReserve_Succeeds`,
  `TestMultiProduct_Isolation`) added together.
- `74c62ed test,docs(specs): fulfill all spec rules with direct tests` —
  new tests drove the spec traceability tightening.
- `cb324f8 feat(events,ci): align event schema with spec, add lint job` —
  `TestReserveRejected_EmitsEvent` drove the `ReserveRejected`
  event type; `TestReserveItem_RepeatedConcurrencyDeterminism` was
  added as the determinism gate from the Level 3 plan.

`go test -race ./...` is the non-negotiable gate; it is wired into CI
and has passed on every commit on `main`.

## Pre-submission Checklist

- [x] `go vet ./...` clean
- [x] `go test ./...` green
- [x] `go test -race ./...` green
- [x] Specs committed under `specs/`
- [x] README quickstart verified on a clean checkout
- [x] Signed commits

## AI Usage Disclosure

- **Tools used**: GitHub Copilot in agent mode.
- **Where AI helped most**:
  - Drafting the planning prompt in `docs/prompt.md`.
  - Scaffolding the layered project structure, repository, lock manager,
    clock abstraction, and test table layout.
  - Drafting spec files using a consistent rule/test/traceability format.
- **Reviewed / rewritten by hand**:
  - Lifecycle invariants (`Confirm` must observe expiry window; terminal
    states cannot transition).
  - Locking discipline: per-product lock wraps *every* counter mutation.
  - Concurrency stress test expectations (`1 success, N-1 OutOfStock`).
- **Rejected**:
  - Optimistic retry with atomic counters. Rejected to keep the locking
    story auditable for review.
  - Background timer goroutine for expiry. Rejected in favour of lazy
    expiry inside `ReserveItem` + an explicit sweep API.
- **Verification independent of AI**:
  - Ran `go test -race ./...` locally.
  - Read the generated diff for every file before committing.
  - Walked the spec rules against each service method by hand.
