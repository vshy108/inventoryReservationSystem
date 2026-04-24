package domain

import "time"

// InventoryEventType is the kind of lifecycle transition recorded in the event log.
type InventoryEventType string

const (
	EventReserved        InventoryEventType = "Reserved"
	EventReserveRejected InventoryEventType = "ReserveRejected"
	EventConfirmed       InventoryEventType = "Confirmed"
	EventCancelled       InventoryEventType = "Cancelled"
	EventExpired         InventoryEventType = "Expired"
)

// InventoryEvent is an immutable record of a reservation lifecycle event.
// Events are append-only and ordered by insertion. ReservationID, UserID,
// and Reason are pointers because not every event carries them (for
// example, a ReserveRejected event has no reservation and may carry a
// reason string).
type InventoryEvent struct {
	EventID       string
	Type          InventoryEventType
	ProductID     string
	ReservationID *string
	UserID        *string
	Reason        *string
	OccurredAt    time.Time
}
