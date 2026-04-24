# 06 — Concurrency (Level 3)

**Spec ID prefix:** `CO`

## Purpose
Guarantee that under high contention the `Available` invariant
(`SF-R2: Available >= 0`) is never violated and exactly the correct number
of reservations succeed.

## Strategy
Per-product `sync.Mutex` acquired around the read-check-write sequence in
`ReserveItem`, `ConfirmReservation`, `CancelReservation`, and
`ExpireReservations`. Per-product locking means reservations on different
products scale in parallel; contention is localized to the hot SKU.

## Rules (normative)
- **CO-R1**: Only one goroutine at a time may execute the critical section
  `read product -> check available -> write reservation + counters` for a
  given product.
- **CO-R2**: All counter mutations for a product must happen inside the
  per-product lock.
- **CO-R3**: `go test -race ./...` must pass on all provided tests.
- **CO-R4**: For any product with `TotalStock = N` and `K > N` concurrent
  reserve attempts, exactly `N` succeed and `K - N` fail with
  `ErrOutOfStock`. No negative `Available` is ever observable.

## Acceptance scenarios
- **CO-S1**: `TotalStock = 1`, 500 goroutines call `ReserveItem`
  concurrently -> exactly 1 success and 499 `ErrOutOfStock` failures.
- **CO-S2**: After CO-S1, `Available = 0`.
- **CO-S3**: The race detector reports no data races.

## Traceability
| Rule  | Test(s) | Implementation |
|-------|---------|----------------|
| CO-R1, CO-R2 | `TestReserveItem_500Concurrent_Stock1` | `LockManager.With`, `ReservationService.ReserveItem` |
| CO-R3 | `go test -race ./...` (CI gate) | `internal/infrastructure/lock_manager.go` |
| CO-R4 | `TestReserveItem_500Concurrent_Stock1` | `ReservationService.ReserveItem` |

## Out of scope
- Optimistic concurrency / retry loops (simpler mutex strategy chosen for
  auditability).
- Distributed locking across processes (single-process in-memory only).
