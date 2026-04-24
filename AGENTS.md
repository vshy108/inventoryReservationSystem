# Agent Instructions — Inventory Reservation System

House rules for any AI coding agent (Copilot, Cursor, etc.) or human
contributor. Read this before making changes.

## 1. Language & Runtime
- **Go** (pinned to 1.23 in `go.mod` and CI).
- Module path: `everest/inventoryReservation`.

## 2. Architecture
Layered, dependencies point **inward only**:

```
interface  ->  application  ->  domain
                    ^
              infrastructure
```

- `internal/domain/`: pure types, sentinel errors, no I/O, no `time.Now()`.
- `internal/application/`: services orchestrating the domain. Depends on
  repository + clock types from `internal/infrastructure/`.
- `internal/infrastructure/`: in-memory repository, per-product
  `LockManager`, `Clock` abstraction (`SystemClock`, `FakeClock`).
- `cmd/app/`: thin entrypoint/demo.

## 3. Core Conventions
- **Errors**: services return sentinel errors from `internal/domain/errors.go`.
  Use `errors.Is` at call sites. Reserve `panic` for true programmer errors.
- **Clock**: inject `infrastructure.Clock`. Never call `time.Now()` in
  `domain/` or `application/`.
- **Concurrency**: all read-check-write sequences on a product go through
  `LockManager.With(productID, fn)`. No ad-hoc locking.
- **Repository**: callers must hold the per-product lock before calling
  `UpdateProduct` / `UpdateReservation` when the mutation must be
  consistent with a prior read.
- **State machine**: terminal states (`Confirmed`, `Cancelled`, `Expired`)
  never transition.
- **IDs**: generated via an atomic counter; treated as opaque strings.

## 4. Testing
- Standard `testing` package. No external test deps.
- Test names mirror spec IDs where useful
  (e.g. `TestReserveItem_500Concurrent_Stock1` for `CO-S1`).
- No real network, filesystem, or wall clock in tests — use `FakeClock`.
- Every concurrency-sensitive test must pass under `go test -race`.
- Red -> green -> refactor. Commit after green.

## 5. Error Handling
- Validate inputs at boundaries (the `interface/` or `cmd/` layer if added).
- Domain errors are a closed set in `internal/domain/errors.go`.
- Never swallow errors.

## 6. Commits
- Conventional Commits: `feat`, `fix`, `test`, `docs`, `refactor`, `chore`, `ci`.
- Scope where useful: `feat(reservation): ...`, `test(concurrency): ...`.
- Each commit should map to one or more spec IDs where possible
  (e.g. `CO-R1`, `RL-R6`).
- Commits should be **signed** (`git commit -S`).

## 7. Security / Secure Coding
- No secrets in code or history.
- Input validation at boundaries before data reaches the service layer.
- Keep `go vet ./...` clean.

## 8. Verification (must be green before push)
```sh
go vet ./...
go test ./...
go test -race ./...
```
