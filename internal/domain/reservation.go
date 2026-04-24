package domain

import "time"

// ReservationState represents the lifecycle state of a reservation.
type ReservationState string

const (
	StateActive    ReservationState = "Active"
	StateConfirmed ReservationState = "Confirmed"
	StateCancelled ReservationState = "Cancelled"
	StateExpired   ReservationState = "Expired"
)

// Reservation is a temporary hold on one unit of a product for a user.
type Reservation struct {
	ReservationID string
	ProductID     string
	UserID        string
	State         ReservationState
	CreatedAt     time.Time
	ExpiresAt     time.Time
	ConfirmedAt   *time.Time
	CancelledAt   *time.Time
	ExpiredAt     *time.Time
}

// IsActive returns true when the reservation is still a live hold.
func (r Reservation) IsActive() bool { return r.State == StateActive }
