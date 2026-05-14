# Inventory Reservation System Cheatsheet

## Commands

```sh
go mod tidy
go test ./...
go test -race ./...
make vet
make test
make race
make cover
make demo
make server
```

Run the HTTP service:

```sh
go run ./cmd/server
go run ./cmd/server --addr=:9000 --seed="sku-1:5,sku-2:2" --hold=2m
```

## HTTP Endpoints

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/reservations` | Reserve one unit |
| `POST` | `/reservations/{id}/confirm` | Confirm a reservation |
| `POST` | `/reservations/{id}/cancel` | Cancel a reservation |
| `GET` | `/reservations/{id}` | Fetch a reservation |
| `GET` | `/products/{id}/stock` | Read available stock |
| `GET` | `/healthz` | Liveness probe |

## Core Invariants

- A reservation starts `Active` and can move to `Confirmed`, `Cancelled`, or `Expired`.
- Confirmed reservations are final and do not return stock.
- Cancelled and expired reservations release stock.
- `ReserveItem` lazily expires stale active reservations before checking availability.
- Per-product locking protects the read-check-write stock mutation.
- Different products should not contend on the same lock.

## Architecture Boundary

```text
interface -> application -> domain
                 ^
           infrastructure
```

- `internal/domain/`: pure reservation and inventory state.
- `internal/application/`: lifecycle orchestration and locking discipline.
- `internal/infrastructure/`: in-memory repository, lock manager, clocks.
- `internal/interface/http/`: HTTP adapter and error mapping.

## Review Gates

```sh
go vet ./...
go test ./...
go test -race ./...
make cover
```
