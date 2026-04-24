# Glossary

Canonical definitions used across all specs. If a term is not here, it is
not part of the domain.

## Actors

### User
The caller reserving, confirming, or cancelling a product. Identified by
`UserID` passed explicitly to service methods. There is no session or
authentication in this challenge.

## Entities

### Product
A sellable SKU with a fixed `TotalStock`. Identified by `ProductID`.

### ProductInventory
Per-product counters:
- `TotalStock`: total units ever available.
- `ConfirmedCount`: units that have been purchased (final).
- `ActiveReservationCount`: units currently held by active reservations.

### Reservation
A temporary hold on exactly one unit of a product for one user. Fields:
- `ReservationID`, `ProductID`, `UserID`
- `State`: `Active | Confirmed | Cancelled | Expired`
- `CreatedAt`, `ExpiresAt`
- Terminal timestamps: `ConfirmedAt`, `CancelledAt`, `ExpiredAt`.

## Concepts

### Available Stock
`Available = TotalStock - ConfirmedCount - ActiveReservationCount`. Must
always be `>= 0`.

### Hold Duration
The lifetime of an active reservation. Default: **2 minutes**.

### Terminal State
`Confirmed`, `Cancelled`, `Expired`. No further transitions are permitted
from a terminal state.

## Out of scope (explicit non-terms)
- Authentication, sessions, tokens.
- Partial reservations or multi-unit reservations (each reservation holds 1 unit).
- Payment, pricing, currency.
- Persistence beyond in-memory (a repository port allows swapping later).
