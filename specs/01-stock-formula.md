# 01 — Stock Formula

**Spec ID prefix:** `SF`

## Purpose
Define how available stock is derived from a `ProductInventory`.

## Rules (normative)
- **SF-R1**: `Available(p) = p.TotalStock - p.ConfirmedCount - p.ActiveReservationCount`.
- **SF-R2**: `Available(p) >= 0` must hold at all times.
- **SF-R3**: `p.ConfirmedCount >= 0` and `p.ActiveReservationCount >= 0`
  must hold at all times.
- **SF-R4**: `p.TotalStock` is fixed for the lifetime of the product in
  this challenge (no restocking).

## Invariants
- **SF-I1**: Any operation that would violate SF-R2 or SF-R3 must be
  rejected before the counters are mutated.

## Acceptance scenarios
- **SF-S1**: Given `TotalStock=5, Confirmed=1, Active=2`, Then
  `Available = 2`.
- **SF-S2**: Given `TotalStock=1, Confirmed=0, Active=1`, Then
  `Available = 0`.
- **SF-S3**: Given `TotalStock=1, Confirmed=1, Active=0`, Then
  `Available = 0`.

## Traceability
| Rule  | Test(s) | Implementation |
|-------|---------|----------------|
| SF-R1 | `TestAvailableStock_Formula` (SF-S1/S2/S3), `TestReserveItem_*`, `TestCancelReservation_ReleasesStock` | `internal/domain/inventory.go: AvailableStock` |
| SF-R2 | `TestReserveItem_FailsWhenStockUnavailable`, `TestReserveItem_500Concurrent_Stock1` | `internal/application/reservation_service.go: ReserveItem` |
| SF-R3 | Enforced by SF-R2 + RL-R2..R4 (counters only decrement from positive) | `ReservationService.*Reservation` |
| SF-R4 | No restock API exists | `internal/domain/inventory.go` |
