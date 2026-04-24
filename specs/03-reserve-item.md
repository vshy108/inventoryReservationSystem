# 03 — Reserve Item (Level 1)

**Spec ID prefix:** `RI`

## Purpose
Create a one-unit temporary hold on a product for a user.

## Signature
```go
ReserveItem(ctx context.Context, productID, userID string) (Reservation, error)
```

## Rules (normative)
- **RI-R1**: If the product does not exist, return `ErrProductNotFound`.
- **RI-R2**: If `Available(product) <= 0`, return `ErrOutOfStock` and
  mutate nothing.
- **RI-R3**: Otherwise, create a new `Active` reservation with:
  - `CreatedAt = clock.Now()`
  - `ExpiresAt = CreatedAt + HoldDuration`
  - increment `ActiveReservationCount` by 1.
- **RI-R4**: Stale active reservations for the product whose
  `ExpiresAt <= now` are expired lazily before the availability check, so
  `Available` reflects current reality.

## Errors
- **RI-E1** `ErrProductNotFound`
- **RI-E2** `ErrOutOfStock`

## Acceptance scenarios
- **RI-S1**: Stock = 1, User A reserves -> success (state `Active`,
  available becomes 0).
- **RI-S2**: Stock = 1, User A reserves, User B reserves -> B gets
  `ErrOutOfStock`.
- **RI-S3**: Unknown product -> `ErrProductNotFound`.

## Traceability
| Rule  | Test(s) | Implementation |
|-------|---------|----------------|
| RI-R1 | `TestReserveItem_UnknownProduct` | `ReservationService.ReserveItem` |
| RI-R2 | `TestReserveItem_FailsWhenStockUnavailable` | `ReservationService.ReserveItem` |
| RI-R3 | `TestReserveItem_SucceedsWhenStockAvailable` | `ReservationService.ReserveItem` |
| RI-R4 | `TestReserveAfterExpiry_FreesSlot` | `ReservationService.expireForProductLocked` |
