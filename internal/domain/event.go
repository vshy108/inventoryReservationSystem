package domain

import "time"

// InventoryEventType is the kind of lifecycle transition recorded in the event log.
type InventoryEventType string

const (
	EventReserved  InventoryEventType = "Reserved"
	EventConfirmed InventoryEventType = "Confirmed"
	EventCancelled InventoryEventType = "Cancelled"
	EventExpired   InventoryEventType = "Expired"
)

// InventoryEvent is an immutable record of a reservation state change.
// Events are append-only and ordered by OccurredAt / insertion.
type InventoryEvent struct {
	Type          InventoryEventType
	ReservationID string
	ProductID     string
	UserID        string
	OccurredAt    time.Time
}
