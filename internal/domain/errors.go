package domain

import "errors"

// Sentinel errors for the reservation domain.
var (
	ErrOutOfStock                  = errors.New("out of stock")
	ErrReservationNotFound         = errors.New("reservation not found")
	ErrReservationAlreadyFinalized = errors.New("reservation already finalized")
	ErrReservationExpired          = errors.New("reservation expired")
	ErrProductNotFound             = errors.New("product not found")
)
