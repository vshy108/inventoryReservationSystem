# Persistence Decision

## Status

Accepted: keep the current implementation in memory for this repo slice.

## Context

The original brief asks for an in-memory inventory reservation system that demonstrates stock accounting, reservation lifecycle transitions, expiry, and safe behavior under high contention. Those behaviors are already covered by specs, focused tests, race tests, HTTP examples, metrics, and the contention demo.

Adding a durable store now would introduce transaction isolation, migration, schema evolution, and process coordination questions that are useful topics, but they would be larger than the current learning objective.

## Decision

Do not add a PostgreSQL or other durable repository adapter yet. Keep `internal/infrastructure/InMemoryRepository` as the explicit storage boundary for this repo state.

Before a durable adapter is added, define a repository contract that covers:

- atomic product counter updates under contention
- reservation creation and state transitions in one transaction boundary
- event append behavior for accepted and rejected operations
- expiry sweeps that cannot double-release a hold
- migration tests that prove the schema enforces the stock and lifecycle invariants

## Consequences

The repo stays small and reviewable for the concurrency and lifecycle brief. The service is not process-durable: restarting the server loses seeded products, reservations, and events. That limitation is intentional and should remain visible in README and cheatsheet references.

A future persistence slice should start with contract tests and migrations before implementing an adapter. It should not swap storage behind the service first and hope the existing in-memory tests are enough.