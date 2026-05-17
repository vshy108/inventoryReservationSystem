# Inventory Reservation System Improvement Plan

This plan captures small, verifiable improvements for the Go reservation service. Keep the layered architecture intact and run the race detector for every behavior change.

## S1 — HTTP Contract Examples

- [x] Add runnable examples for reserve, confirm, cancel, stock lookup, and liveness endpoints.
- [x] Include expected status codes for unknown products, out-of-stock reservations, finalized reservations, and malformed requests.
- [x] Verify with: `go test ./...` and a local `go run ./cmd/server` smoke path.

## S2 — Boundary Validation Matrix

- [x] Document request validation rules at the HTTP boundary before inputs reach the application service.
- [x] Add tests for invalid product IDs, invalid reservation IDs, and unsupported payload shapes.
- [x] Verify with: `make test`.

## S3 — Expiry Sweeper Observability

- [x] Add lightweight logs or counters for explicit expiry sweeps without changing domain purity.
- [x] Document how lazy expiry and explicit sweep interact.
- [x] Verify with focused expiry tests plus `go test -race ./...`.

## S4 — Load And Contention Demo

- [x] Add a small reviewer-facing load script or command that demonstrates single-winner behavior under contention.
- [x] Record the expected success/out-of-stock shape in the README or cheatsheet.
- [x] Verify with: `make race`.

## S5 — Persistence Spike Decision

- [x] Write a short ADR deciding whether a durable repository is worth adding beyond the current in-memory brief.
- [x] If yes, define the repository contract and migration tests before implementing an adapter.
- [x] If no, document the reason and keep the in-memory design explicit.

## S6 — Shadow Persistence Migration Contract

- [x] Add additive PostgreSQL schema migrations for `inventory_items`, `reservations`, `idempotency_keys`, and observe-only `outbox_events`.
- [x] Keep the in-memory repository as the command source of truth; do not wire a durable adapter in this slice.
- [x] Verify the migration package with `go test ./...` and a Docker-backed apply/drop smoke using `scripts/postgres_migration_smoke.sh`.

## S7 — Shadow Recorder Application Seam

- [x] Add an application-facing `ShadowRecorder` interface with a no-op implementation for default service construction.
- [x] Emit observe-only shadow records after successful reserve, confirm, cancel, and expire transitions without changing command responses.
- [x] Verify with focused recorder tests plus `go vet ./...`, `go test ./...`, and `go test -race ./...`.

## S8 — OpenAPI 3.0 Contract

- [x] Write an OpenAPI 3.0 spec covering all HTTP endpoints: `POST /reservations`, confirm, cancel, `GET /reservations/{id}`, `GET /products/{id}/stock`, `GET /healthz`, and `GET /openapi.yaml` itself.
- [x] Embed the spec in the handler package (`go:embed`) and serve it at `GET /openapi.yaml`.
- [x] Update README proof map with a link to the spec file.
- [x] Verify with: `go build ./...` and `curl http://localhost:8080/openapi.yaml` after `go run ./cmd/server`.

## S9 — k6 Load Test Evidence

- [x] Add a k6 script (`k6/reservation_load.js`) targeting `POST /reservations` against the Postgres-backed compose stack.
- [x] Add a compose override (`docker-compose.loadtest.yml`) seeding `sku-load:1000000` so stock does not exhaust during the test.
- [x] Add a driver script (`scripts/k6_load_test.sh`) that starts compose, runs k6, saves a Markdown report to `docs/k6-load-report.md`, and tears down.
- [x] Verify: `bash scripts/k6_load_test.sh` passes all thresholds — p(95) < 500 ms, error rate < 1 %, reservation_created > 0.
- Results (developer laptop, Docker compose): **8 544 reservations in 45 s · 189.9 req/s · avg 113 ms · p(95) 359 ms · 0.00 % errors**.
