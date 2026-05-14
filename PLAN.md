# Inventory Reservation System Improvement Plan

This plan captures small, verifiable improvements for the Go reservation service. Keep the layered architecture intact and run the race detector for every behavior change.

## S1 — HTTP Contract Examples

- [ ] Add runnable examples for reserve, confirm, cancel, stock lookup, and liveness endpoints.
- [ ] Include expected status codes for unknown products, out-of-stock reservations, finalized reservations, and malformed requests.
- [ ] Verify with: `go test ./...` and a local `go run ./cmd/server` smoke path.

## S2 — Boundary Validation Matrix

- [ ] Document request validation rules at the HTTP boundary before inputs reach the application service.
- [ ] Add tests for invalid product IDs, invalid reservation IDs, and unsupported payload shapes.
- [ ] Verify with: `make test`.

## S3 — Expiry Sweeper Observability

- [ ] Add lightweight logs or counters for explicit expiry sweeps without changing domain purity.
- [ ] Document how lazy expiry and explicit sweep interact.
- [ ] Verify with focused expiry tests plus `go test -race ./...`.

## S4 — Load And Contention Demo

- [ ] Add a small reviewer-facing load script or command that demonstrates single-winner behavior under contention.
- [ ] Record the expected success/out-of-stock shape in the README or cheatsheet.
- [ ] Verify with: `make race`.

## S5 — Persistence Spike Decision

- [ ] Write a short ADR deciding whether a durable repository is worth adding beyond the current in-memory brief.
- [ ] If yes, define the repository contract and migration tests before implementing an adapter.
- [ ] If no, document the reason and keep the in-memory design explicit.
