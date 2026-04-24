# 02 — Reservation Lifecycle

**Spec ID prefix:** `RL`

## Purpose
Define the state machine for a `Reservation` and the effects of each
transition on `ProductInventory`.

## States
`Active | Confirmed | Cancelled | Expired`.

## Rules (normative)
- **RL-R1**: A newly created reservation starts in `Active` and increments
  `ActiveReservationCount` by 1.
- **RL-R2**: `Active -> Confirmed`: decrement `ActiveReservationCount`,
  increment `ConfirmedCount`, set `ConfirmedAt`.
- **RL-R3**: `Active -> Cancelled`: decrement `ActiveReservationCount`,
  set `CancelledAt`.
- **RL-R4**: `Active -> Expired`: decrement `ActiveReservationCount`,
  set `ExpiredAt`.
- **RL-R5**: Terminal states (`Confirmed`, `Cancelled`, `Expired`) cannot
  transition. Any such attempt returns an error and mutates nothing.
- **RL-R6**: `Confirm` performed when `now >= ExpiresAt` is treated as an
  expiry: the reservation transitions to `Expired` and the request returns
  `ErrReservationExpired`.

## Errors
- **RL-E1** `ErrReservationNotFound`: reservation id unknown.
- **RL-E2** `ErrReservationAlreadyFinalized`: reservation is in a terminal
  state other than `Expired` (for expired, use RL-E3).
- **RL-E3** `ErrReservationExpired`: reservation has expired (either
  already `Expired`, or `Active` with `now >= ExpiresAt`).

## Acceptance scenarios
- **RL-S1**: Confirm of an `Active` reservation before expiry -> state
  becomes `Confirmed`, `Available` unchanged (hold becomes sale).
- **RL-S2**: Confirm of a `Confirmed` reservation -> `ErrReservationAlreadyFinalized`.
- **RL-S3**: Cancel of an `Active` reservation -> state becomes `Cancelled`,
  `Available` increases by 1.
- **RL-S4**: Cancel of a non-Active reservation -> `ErrReservationAlreadyFinalized`.
- **RL-S5**: Confirm after expiry window -> reservation transitions to
  `Expired` and `ErrReservationExpired` is returned.

## Traceability
| Rule  | Test(s) | Implementation |
|-------|---------|----------------|
| RL-R1 | `TestReserveItem_SucceedsWhenStockAvailable` | `ReservationService.ReserveItem` |
| RL-R2 | `TestConfirmReservation_Finalizes` | `ReservationService.ConfirmReservation` |
| RL-R3 | `TestCancelReservation_ReleasesStock` | `ReservationService.CancelReservation` |
| RL-R4 | `TestExpireReservations_ReleasesExpiredStock` | `ReservationService.ExpireReservations` |
| RL-R5 | `TestConfirmReservation_DoubleConfirm` | `ReservationService.ConfirmReservation` |
| RL-R6 | `TestConfirmAfterExpiry_Fails` | `ReservationService.ConfirmReservation` |
