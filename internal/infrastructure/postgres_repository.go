package infrastructure

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"

	"everest/inventoryReservation/internal/domain"
)

// PostgresRepository persists inventory state to a PostgreSQL database.
// It is wired in when DATABASE_URL is present; otherwise the service falls
// back to InMemoryRepository.
//
// Concurrency model: each UpdateProduct / UpdateReservation call wraps its
// read-check-write in a transaction with SELECT FOR UPDATE so that concurrent
// processes (e.g. two replicas) cannot oversell even without the in-process
// LockManager.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository wraps an open *sql.DB as a Repository.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// — Events ——————————————————————————————————————————————————————

// AppendEvent inserts an immutable event into the outbox_events table.
// Duplicate event IDs are silently ignored (ON CONFLICT DO NOTHING) so
// the method is safe to call more than once for the same event.
func (r *PostgresRepository) AppendEvent(e domain.InventoryEvent) {
	payload, _ := json.Marshal(e)
	_, err := r.db.ExecContext(context.Background(), `
		INSERT INTO outbox_events
			(event_id, event_type, product_id, reservation_id, user_id, reason, occurred_at, payload)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (event_id) DO NOTHING`,
		e.EventID, string(e.Type), e.ProductID,
		e.ReservationID, e.UserID, e.Reason,
		e.OccurredAt, payload)
	if err != nil {
		log.Printf("warn: AppendEvent %s: %v", e.EventID, err)
	}
}

// AllEvents returns all recorded events ordered by insertion sequence.
func (r *PostgresRepository) AllEvents() []domain.InventoryEvent {
	rows, err := r.db.QueryContext(context.Background(), `
		SELECT event_id, event_type, product_id, reservation_id, user_id, reason, occurred_at
		FROM outbox_events
		ORDER BY outbox_id`)
	if err != nil {
		log.Printf("warn: AllEvents query: %v", err)
		return nil
	}
	defer rows.Close()

	var events []domain.InventoryEvent
	for rows.Next() {
		var e domain.InventoryEvent
		var eventType string
		if err := rows.Scan(
			&e.EventID, &eventType, &e.ProductID,
			&e.ReservationID, &e.UserID, &e.Reason, &e.OccurredAt,
		); err != nil {
			log.Printf("warn: AllEvents scan: %v", err)
			continue
		}
		e.Type = domain.InventoryEventType(eventType)
		events = append(events, e)
	}
	return events
}

// — Products ——————————————————————————————————————————————————————

// AddProduct upserts a product; existing counters are replaced entirely.
func (r *PostgresRepository) AddProduct(p domain.ProductInventory) {
	_, err := r.db.ExecContext(context.Background(), `
		INSERT INTO inventory_items
			(product_id, total_stock, confirmed_count, active_reservation_count, updated_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (product_id) DO UPDATE SET
			total_stock              = EXCLUDED.total_stock,
			confirmed_count          = EXCLUDED.confirmed_count,
			active_reservation_count = EXCLUDED.active_reservation_count,
			updated_at               = now()`,
		p.ProductID, p.TotalStock, p.ConfirmedCount, p.ActiveReservationCount)
	if err != nil {
		log.Printf("warn: AddProduct %s: %v", p.ProductID, err)
	}
}

// GetProduct returns a snapshot of the product inventory.
func (r *PostgresRepository) GetProduct(id string) (domain.ProductInventory, bool) {
	var p domain.ProductInventory
	err := r.db.QueryRowContext(context.Background(), `
		SELECT product_id, total_stock, confirmed_count, active_reservation_count
		FROM inventory_items
		WHERE product_id = $1`, id).
		Scan(&p.ProductID, &p.TotalStock, &p.ConfirmedCount, &p.ActiveReservationCount)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ProductInventory{}, false
	}
	if err != nil {
		log.Printf("warn: GetProduct %s: %v", id, err)
		return domain.ProductInventory{}, false
	}
	return p, true
}

// UpdateProduct fetches a product under SELECT FOR UPDATE, applies fn, then
// writes the result back — all inside a single transaction.  This ensures
// correctness even when multiple service replicas are running concurrently.
func (r *PostgresRepository) UpdateProduct(id string, fn func(*domain.ProductInventory) error) error {
	ctx := context.Background()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	var p domain.ProductInventory
	err = tx.QueryRowContext(ctx, `
		SELECT product_id, total_stock, confirmed_count, active_reservation_count
		FROM inventory_items
		WHERE product_id = $1
		FOR UPDATE`, id).
		Scan(&p.ProductID, &p.TotalStock, &p.ConfirmedCount, &p.ActiveReservationCount)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrProductNotFound
	}
	if err != nil {
		return err
	}

	if err := fn(&p); err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE inventory_items
		SET total_stock=$2, confirmed_count=$3, active_reservation_count=$4, updated_at=now()
		WHERE product_id=$1`,
		p.ProductID, p.TotalStock, p.ConfirmedCount, p.ActiveReservationCount)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// — Reservations ——————————————————————————————————————————————————

// SaveReservation upserts a reservation row.  Called both on initial insert
// (StateActive) and on state transitions (Confirmed / Cancelled / Expired).
func (r *PostgresRepository) SaveReservation(res domain.Reservation) {
	_, err := r.db.ExecContext(context.Background(), `
		INSERT INTO reservations
			(reservation_id, product_id, user_id, state,
			 created_at, expires_at, confirmed_at, cancelled_at, expired_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now())
		ON CONFLICT (reservation_id) DO UPDATE SET
			state        = EXCLUDED.state,
			confirmed_at = EXCLUDED.confirmed_at,
			cancelled_at = EXCLUDED.cancelled_at,
			expired_at   = EXCLUDED.expired_at,
			updated_at   = now()`,
		res.ReservationID, res.ProductID, res.UserID, string(res.State),
		res.CreatedAt, res.ExpiresAt, res.ConfirmedAt, res.CancelledAt, res.ExpiredAt)
	if err != nil {
		log.Printf("warn: SaveReservation %s: %v", res.ReservationID, err)
	}
}

// GetReservation returns a snapshot of a reservation by ID.
func (r *PostgresRepository) GetReservation(id string) (domain.Reservation, bool) {
	var res domain.Reservation
	var state string
	err := r.db.QueryRowContext(context.Background(), `
		SELECT reservation_id, product_id, user_id, state,
		       created_at, expires_at, confirmed_at, cancelled_at, expired_at
		FROM reservations
		WHERE reservation_id = $1`, id).
		Scan(&res.ReservationID, &res.ProductID, &res.UserID, &state,
			&res.CreatedAt, &res.ExpiresAt, &res.ConfirmedAt, &res.CancelledAt, &res.ExpiredAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Reservation{}, false
	}
	if err != nil {
		log.Printf("warn: GetReservation %s: %v", id, err)
		return domain.Reservation{}, false
	}
	res.State = domain.ReservationState(state)
	return res, true
}

// UpdateReservation fetches a reservation under SELECT FOR UPDATE, applies fn,
// then writes the state transition back — all inside a single transaction.
func (r *PostgresRepository) UpdateReservation(id string, fn func(*domain.Reservation) error) error {
	ctx := context.Background()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	var res domain.Reservation
	var state string
	err = tx.QueryRowContext(ctx, `
		SELECT reservation_id, product_id, user_id, state,
		       created_at, expires_at, confirmed_at, cancelled_at, expired_at
		FROM reservations
		WHERE reservation_id = $1
		FOR UPDATE`, id).
		Scan(&res.ReservationID, &res.ProductID, &res.UserID, &state,
			&res.CreatedAt, &res.ExpiresAt, &res.ConfirmedAt, &res.CancelledAt, &res.ExpiredAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrReservationNotFound
	}
	if err != nil {
		return err
	}
	res.State = domain.ReservationState(state)

	if err := fn(&res); err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE reservations
		SET state=$2, confirmed_at=$3, cancelled_at=$4, expired_at=$5, updated_at=now()
		WHERE reservation_id=$1`,
		res.ReservationID, string(res.State), res.ConfirmedAt, res.CancelledAt, res.ExpiredAt)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// ActiveReservationsByProduct returns all Active reservations for a product.
func (r *PostgresRepository) ActiveReservationsByProduct(productID string) []domain.Reservation {
	rows, err := r.db.QueryContext(context.Background(), `
		SELECT reservation_id, product_id, user_id, state,
		       created_at, expires_at, confirmed_at, cancelled_at, expired_at
		FROM reservations
		WHERE product_id = $1 AND state = 'Active'`, productID)
	if err != nil {
		log.Printf("warn: ActiveReservationsByProduct %s: %v", productID, err)
		return nil
	}
	defer rows.Close()
	return scanReservations(rows)
}

// AllActiveReservations returns every reservation currently in the Active state.
func (r *PostgresRepository) AllActiveReservations() []domain.Reservation {
	rows, err := r.db.QueryContext(context.Background(), `
		SELECT reservation_id, product_id, user_id, state,
		       created_at, expires_at, confirmed_at, cancelled_at, expired_at
		FROM reservations
		WHERE state = 'Active'`)
	if err != nil {
		log.Printf("warn: AllActiveReservations query: %v", err)
		return nil
	}
	defer rows.Close()
	return scanReservations(rows)
}

// scanReservations reads all rows from a reservation query result.
func scanReservations(rows *sql.Rows) []domain.Reservation {
	var out []domain.Reservation
	for rows.Next() {
		var res domain.Reservation
		var state string
		if err := rows.Scan(
			&res.ReservationID, &res.ProductID, &res.UserID, &state,
			&res.CreatedAt, &res.ExpiresAt, &res.ConfirmedAt, &res.CancelledAt, &res.ExpiredAt,
		); err != nil {
			log.Printf("warn: scanReservations: %v", err)
			continue
		}
		res.State = domain.ReservationState(state)
		out = append(out, res)
	}
	return out
}
