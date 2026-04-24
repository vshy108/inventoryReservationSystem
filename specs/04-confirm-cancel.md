# 04 — Confirm & Cancel (Level 2)

**Spec ID prefix:** `CC`

## Purpose
Transition an `Active` reservation to a terminal state on explicit user
action.

## Signatures
```go
ConfirmReservation(ctx context.Context, reservationID string) (Reservation, error)
CancelReservation(ctx context.Context, reservationID string) (Reservation, error)
```

## Rules (normative)
- **CC-R1**: Both operations acquire the per-product lock for the
  reservation's product to serialize counter updates.
- **CC-R2**: `Confirm` when `State == Active` and `now < ExpiresAt`:
  transition to `Confirmed`, decrement `ActiveReservationCount`, increment
  `ConfirmedCount`, set `ConfirmedAt`.
- **CC-R3**: `Confirm` when `State == Active` and `now >= ExpiresAt`:
  treat as expiry (RL-R6), return `ErrReservationExpired`.
- **CC-R4**: `Confirm` when `State != Active`:
  - If `Expired` -> `ErrReservationExpired`.
  - Else -> `ErrReservationAlreadyFinalized`.
- **CC-R5**: `Cancel` when `State == Active`: transition to `Cancelled`,
  decrement `ActiveReservationCount`, set `CancelledAt`.
- **CC-R6**: `Cancel` when `State != Active` -> `ErrReservationAlreadyFinalized`.
- **CC-R7**: Confirmed reservations are final and can never transition
  (aligned with the brief: confirmed purchases cannot be reversed).

## Errors
- **CC-E1** `ErrReservationNotFound`
- **CC-E2** `ErrReservationAlreadyFinalized`
- **CC-E3** `ErrReservationExpired`

## Acceptance scenarios
- **CC-S1**: Confirm active reservation -> state `Confirmed`; `Available`
  unchanged.
- **CC-S2**: Confirm already-confirmed reservation -> `ErrReservationAlreadyFinalized`.
- **CC-S3**: Cancel active reservation -> state `Cancelled`; `Available`
  increases by 1.
- **CC-S4**: Confirm after expiry window -> `ErrReservationExpired` and
  reservation becomes `Expired`.

## Traceability
| Rule  | Test(s) | Implementation |
|-------|---------|----------------|
| CC-R1 | `TestReserveItem_500Concurrent_Stock1` (race-tested) | `LockManager.With` |
| CC-R2 | `TestConfirmReservation_Finalizes` | `ReservationService.ConfirmReservation` |
| CC-R3 | `TestConfirmAfterExpiry_Fails` | `ReservationService.ConfirmReservation` |
| CC-R4 | `TestConfirmReservation_DoubleConfirm`, `TestConfirmCancel_UnknownID` | `ReservationService.ConfirmReservation` |
| CC-R5 | `TestCancelReservation_ReleasesStock` | `ReservationService.CancelReservation` |
| CC-R6 | `TestCancel_Confirmed_Fails`, `TestCancelThenReReserve_Succeeds` | `ReservationService.CancelReservation` |
| CC-R7 | `TestConfirmReservation_DoubleConfirm` | `ReservationService.ConfirmReservation` |
