# Plan — Inventory Reservation System (Flash Sale Concurrency Challenge)

## 1. Problem Summary
Build an **Inventory Reservation System** that prevents overselling during high-concurrency flash sales.

Language target: **Go** (Golang), with in-memory state and explicit concurrency control.

Core scenario from the challenge:
- Stock can be very limited (for example, 1 item).
- Many users can try to reserve the same item at nearly the same time (for example, 500 requests).
- The system must allow at most one successful reservation for the last item and reject the rest.

Primary objectives:
- Prevent overselling.
- Handle concurrent requests safely.
- Manage temporary reservations with expiry.
- Maintain a consistent inventory state at all times.

---

## 2. Business Rules
- Available stock formula:
  - **Available Stock = Total Stock - Confirmed Sales - Active Reservations**
- Reservation requests that exceed available stock must fail.
- Confirmed purchases are final and cannot be reversed.
- Only one user can reserve the last item.
- Reservations automatically release inventory when expired.
- Reservation hold time: **2 minutes**.

---

## 3. Deliverables by Level

### Level 1 — Basic Inventory Reservation
- Keep inventory state in memory.
- Implement `ReserveItem(ctx, productID, userID)`.
- Reject reservation when no stock is available.
- Example acceptance check:
  - Stock = 1
  - User A reserve -> success
  - User B reserve -> fail

### Level 2 — Reservation Lifecycle & Expiry
- Add reservation states:
  - `Active`
  - `Confirmed`
  - `Cancelled`
  - `Expired`
- Add `ConfirmReservation(ctx, reservationID)` transition:
  - Converts an active hold into a confirmed sale.
- Add `CancelReservation(ctx, reservationID)` transition:
  - Releases reserved inventory.
- Implement automatic expiry (2-minute hold):
  - Expired reservations release inventory.

### Level 3 — Concurrency Handling
- Prevent race conditions under simultaneous requests.
- Validate with scenario:
  - Stock = 1
  - Simultaneous requests = 500
  - Expected: 1 success, 499 failures
- Use an explicit thread-safety strategy:
  - Mutex/locks per product, or
  - Atomic transactional update with optimistic retry, or
  - Single-writer queue per SKU.

---

## 4. Domain Model

```text
ProductInventory struct {
  ProductID string
  TotalStock int
  ConfirmedCount int
  ActiveReservationCount int
}

Reservation struct {
  ReservationID string
  ProductID string
  UserID string
  State ReservationState // Active|Confirmed|Cancelled|Expired
  CreatedAt time.Time
  ExpiresAt time.Time
  ConfirmedAt *time.Time
  CancelledAt *time.Time
  ExpiredAt *time.Time
}

InventoryEvent struct {
  EventID string
  ReservationID *string
  ProductID string
  UserID *string
  Type EventType // Reserved|ReserveRejected|Confirmed|Cancelled|Expired
  Timestamp time.Time
  Reason *string
}
```

### Invariants
- `availableStock >= 0` must always hold.
- `activeReservationCount >= 0` and `confirmedCount >= 0`.
- A reservation can only move through valid transitions:
  - `Active -> Confirmed | Cancelled | Expired`
  - Terminal states (`Confirmed`, `Cancelled`, `Expired`) cannot transition.
- Confirm action only valid for `Active` reservation before expiry.

---

## 5. Go Service API (Suggested)

```text
type ReservationService interface {
  ReserveItem(ctx context.Context, productID, userID string, now time.Time) (Reservation, error)
  ConfirmReservation(ctx context.Context, reservationID string, now time.Time) (Reservation, error)
  CancelReservation(ctx context.Context, reservationID string, now time.Time) (Reservation, error)
  ExpireReservations(ctx context.Context, now time.Time) (int, error)
  GetAvailableStock(ctx context.Context, productID string) (int, error)
  GetReservation(ctx context.Context, reservationID string) (Reservation, bool, error)
}
```

Possible error categories:
- `ErrOutOfStock`
- `ErrReservationNotFound`
- `ErrReservationAlreadyFinalized`
- `ErrReservationExpired`
- `ErrProductNotFound`

---

## 6. Architecture (Clean and Testable)

```text
Interface Layer (CLI / HTTP / Test Harness)
  -> Application Layer (ReservationService, InventoryService)
    -> Domain Layer (entities, policies, state transitions)
      -> Infrastructure Layer (in-memory repositories, lock manager, clock)
```

Key design points:
- Keep domain logic deterministic and side-effect free where possible.
- Abstract current time behind a `Clock` interface for deterministic tests.
- Centralize concurrency control in one place (lock manager or transaction boundary).
- Emit events for every important state transition for auditability.
- Run `go test -race ./...` to validate thread safety assumptions.

---

## 7. Concurrency Strategy

Preferred baseline for this challenge:
- Use **per-product mutex lock** (`sync.Mutex`) around the read-check-write sequence in `ReserveItem`.
- Inside lock:
  1. Recompute available stock from current state.
  2. If available <= 0, reject.
  3. Create active reservation and increment active count.

Why this is enough:
- Avoids check-then-act race.
- Easy to reason about in interviews/reviews.
- Works for the required high-contention single-product test.

Scalability note:
- Per-product locking allows parallel reservations for different products.
- Back the lock map with `sync.Map` or a guarded map to safely manage lock instances.

---

## 8. TDD Plan

### Level 1 tests
- Reserve succeeds when stock available.
- Reserve fails when stock unavailable.
- Available stock calculation follows business formula.

### Level 2 tests
- Confirm active reservation updates counts and state.
- Cancel active reservation releases stock and sets state.
- Expiry job transitions timed-out active reservations to expired.
- Confirm after expiry fails.

### Level 3 tests
- 500 parallel reserve calls for stock 1 -> exactly 1 success.
- No negative available stock under contention.
- Repeated high-concurrency runs remain deterministic in outcome counts.
- Add race detector gate: `go test -race ./...` must pass.

---

## 9. Suggested Project Layout

```text
inventoryReservationSystem/
├── cmd/
│   └── app/
│       └── main.go
├── internal/
│   ├── domain/
│   │   ├── reservation.go
│   │   ├── inventory.go
│   │   ├── errors.go
│   │   └── policies.go
│   ├── application/
│   │   ├── reservation_service.go
│   │   └── inventory_service.go
│   ├── infrastructure/
│   │   ├── in_memory_repository.go
│   │   ├── lock_manager.go
│   │   └── clock.go
│   └── interface/
│       ├── http_handler.go
│       └── dto.go
├── tests/
│   ├── unit/
│   └── concurrency/
├── docs/
│   └── prompt.md
├── go.mod
├── go.sum
└── README.md
```

---

## 10. Implementation Milestones
1. Bootstrap Go module (`go mod init`), add baseline tests, and one passing smoke test.
2. Implement Level 1 reservation + stock formula tests.
3. Implement Level 2 lifecycle transitions + expiry handling.
4. Implement Level 3 locking strategy + parallel stress test.
5. Add README and architecture notes, then final verification with `go test ./...` and `go test -race ./...`.

---

## 11. Evaluation Alignment Checklist
- Correctness:
  - No overselling.
  - Reservation lifecycle rules are respected.
- Concurrency handling:
  - 500 concurrent requests on stock 1 produce exactly 1 success.
- Expiry logic:
  - Timed-out active reservations release stock automatically.
- Code quality:
  - Clear layering, explicit error model, deterministic tests.
  - Locking strategy is documented and simple to audit.

---

## 12. Done Criteria
- All Level 1, 2, and 3 acceptance tests pass.
- Concurrency scenario is automated and repeatable.
- Stock formula holds under all transitions and stress runs.
- Documentation clearly explains lifecycle and locking decisions.
- A reviewer can clone, run tests, and verify in under a few minutes.
- Verification commands are documented and green:
  - `go test ./...`
  - `go test -race ./...`

---

## 13. Submission Requirements (from challenge email, adapted for Go)
The reviewer expects a ZIP (or Drive link) and will judge the code
**as if written by an experienced engineer**. AI usage is allowed and
expected, but does not lower the quality bar.

### Explicit grading values
- **SOLID principles**: apply where they reduce coupling; avoid over-engineering.
- **Test-driven development**: red -> green -> refactor should be visible in commit flow.
- **Clean, readable code**: small focused functions, clear names, no dead code.
- **OOP and design patterns where appropriate**: in Go, favor interfaces at boundaries,
  composition over inheritance, and lightweight patterns (Repository, Policy, Result-like error modeling)
  only when they improve clarity.
- **Reviewer DX**: make the solution easy to run, inspect, and validate quickly.

### Mandatory deliverables in the ZIP
- `README.md` with:
  - One-paragraph problem statement.
  - **Quickstart** (<= 3 commands), for example:
    - `go mod tidy`
    - `go test ./...`
    - `go test -race ./...`
  - Architecture diagram (layered ASCII block is enough).
  - Spec -> test -> code traceability table (if using a `specs/` folder).
  - Level 1 / 2 / 3 feature checklist with status.
  - **Design decisions and trade-offs** section.
  - **AI usage disclosure** section (see below).
- `AGENTS.md` (if present in the repo) for engineering conventions.
- `specs/` directory (if used) that drives implementation.
- `.github/workflows/ci.yml` configured for Go checks (test and race; optionally lint).
- `go.mod` and `go.sum` committed.
- Clean git history with conventional commits where practical.
- Exclude build artifacts from the ZIP (`bin/`, coverage output files, temporary logs).

### AI usage disclosure (recommended)
Add a section to `README.md` titled `## AI Usage Disclosure` covering:
- Tools used (for example, GitHub Copilot in agent mode; model name if desired).
- Where AI helped most (spec drafting, test scaffolding, boilerplate).
- What was reviewed or rewritten by hand (domain rules, invariants, service boundaries).
- What was rejected and why (over-engineered patterns, unsafe shortcuts).
- Verification done independently of AI (tests, race runs, diff review, edge-case checks).

### Pre-submission checklist
- [ ] `go test ./...` succeeds on a fresh clone.
- [ ] `go test -race ./...` is clean.
- [ ] Lint is clean (for example, `golangci-lint run` if configured).
- [ ] CI badge in `README.md` is green.
- [ ] Public APIs have purpose-driven names; no commented-out code.
- [ ] No secrets, tokens, or personal paths committed.
- [ ] README quickstart is verified line by line on a clean machine.
- [ ] AI Usage Disclosure section is present and accurate.
- [ ] ZIP excludes `.git/`, local caches, and generated artifacts unless explicitly requested.
