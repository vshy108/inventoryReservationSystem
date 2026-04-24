# 05 — Expiry (Level 2)

**Spec ID prefix:** `EX`

## Purpose
Automatically release inventory held by reservations whose hold window has
elapsed.

## Signature
```go
ExpireReservations(ctx context.Context, now time.Time) int
```

## Rules (normative)
- **EX-R1**: A reservation is expired when `State == Active` and
  `now >= ExpiresAt`.
- **EX-R2**: Expiry transitions the reservation to `Expired`, decrements
  `ActiveReservationCount`, and sets `ExpiredAt = now`.
- **EX-R3**: Expiry is also applied lazily inside `ReserveItem` for the
  target product (see RI-R4) so a new reservation can reuse a slot held by
  an expired one without waiting for the sweep.
- **EX-R4**: `ExpireReservations` is idempotent: running twice at the
  same `now` expires each eligible reservation exactly once.
- **EX-R5**: Expiry acquires the per-product lock for each product it
  touches so counters remain consistent under concurrency.

## Acceptance scenarios
- **EX-S1**: After `ExpiresAt`, sweep expires the reservation and
  `Available` increases by 1.
- **EX-S2**: Sweep run twice returns 0 the second time.
- **EX-S3**: New `ReserveItem` after the hold window succeeds even without
  an explicit sweep (lazy expiry).

## Traceability
| Rule  | Test(s) | Implementation |
|-------|---------|----------------|
| EX-R1, EX-R2 | `TestExpireReservations_ReleasesExpiredStock` | `ReservationService.ExpireReservations`, `expireReservationLocked` |
| EX-R3 | `TestReserveAfterExpiry_FreesSlot` | `ReservationService.expireForProductLocked` |
| EX-R4 | `TestExpireReservations_Idempotent` | `ReservationService.ExpireReservations` |
| EX-R5 | `TestReserveItem_500Concurrent_Stock1` + `go test -race` | `LockManager.With` |

## Out of scope
- Background timer goroutine. The brief only requires the logic to be
  correct when invoked; scheduling cadence is a deployment concern.
