# Expiry Observability

Reservations can expire in two ways:

| Path | Trigger | What it protects |
|------|---------|------------------|
| Lazy expiry | `ReserveItem` checks stale active holds for the requested product before checking stock | A new reservation can reuse an expired hold immediately, even if the background worker has not run yet |
| Explicit sweep | `cmd/server` runs `ExpireReservations` on `--expiry-interval` | Idle products eventually release expired holds and emit expiry events without waiting for new traffic |

The explicit sweeper records two counters on `/metrics`:

```text
inventory_expiry_sweeps_total
inventory_expired_reservations_total
```

`inventory_expiry_sweeps_total` increments on every explicit sweep, including sweeps that expire zero reservations. `inventory_expired_reservations_total` only increments by the number of reservations transitioned to `Expired` by the explicit worker.

The server still logs nonzero expiry batches:

```text
expired 3 reservation(s)
```

## Verification

```sh
go test -race ./...
make http-smoke
```