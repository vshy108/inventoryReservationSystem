package application

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"everest/inventoryReservation/internal/domain"
	"everest/inventoryReservation/internal/infrastructure"
)

// DefaultHoldDuration is the reservation hold time from the brief.
const DefaultHoldDuration = 2 * time.Minute

// ReservationService implements the reservation lifecycle with a
// per-product mutex that serializes the read-check-write critical
// section required to prevent overselling.
type ReservationService struct {
	repo      *infrastructure.InMemoryRepository
	locks     *infrastructure.LockManager
	clock     infrastructure.Clock
	holdFor   time.Duration
	idCounter atomic.Uint64
}

// NewReservationService wires a service. A zero holdFor falls back to DefaultHoldDuration.
func NewReservationService(
	repo *infrastructure.InMemoryRepository,
	locks *infrastructure.LockManager,
	clock infrastructure.Clock,
	holdFor time.Duration,
) *ReservationService {
	if holdFor == 0 {
		holdFor = DefaultHoldDuration
	}
	return &ReservationService{repo: repo, locks: locks, clock: clock, holdFor: holdFor}
}

func (s *ReservationService) nextID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, s.idCounter.Add(1))
}

// ReserveItem atomically checks stock and creates an active reservation,
// or returns ErrOutOfStock / ErrProductNotFound.
func (s *ReservationService) ReserveItem(_ context.Context, productID, userID string) (domain.Reservation, error) {
	var result domain.Reservation
	var err error
	s.locks.With(productID, func() {
		now := s.clock.Now()
		// Lazily expire stale active reservations so AvailableStock is accurate.
		s.expireForProductLocked(productID, now)

		product, ok := s.repo.GetProduct(productID)
		if !ok {
			err = domain.ErrProductNotFound
			return
		}
		if product.AvailableStock() <= 0 {
			err = domain.ErrOutOfStock
			return
		}
		res := domain.Reservation{
			ReservationID: s.nextID("res"),
			ProductID:     productID,
			UserID:        userID,
			State:         domain.StateActive,
			CreatedAt:     now,
			ExpiresAt:     now.Add(s.holdFor),
		}
		s.repo.SaveReservation(res)
		_ = s.repo.UpdateProduct(productID, func(p *domain.ProductInventory) error {
			p.ActiveReservationCount++
			return nil
		})
		result = res
	})
	return result, err
}

// ConfirmReservation converts an active reservation into a confirmed sale.
func (s *ReservationService) ConfirmReservation(_ context.Context, reservationID string) (domain.Reservation, error) {
	res, ok := s.repo.GetReservation(reservationID)
	if !ok {
		return domain.Reservation{}, domain.ErrReservationNotFound
	}
	var result domain.Reservation
	var err error
	s.locks.With(res.ProductID, func() {
		current, ok := s.repo.GetReservation(reservationID)
		if !ok {
			err = domain.ErrReservationNotFound
			return
		}
		now := s.clock.Now()
		// Treat as expired if it has outlived its hold even if still marked Active.
		if current.State == domain.StateActive && !now.Before(current.ExpiresAt) {
			s.expireReservationLocked(current.ReservationID, now)
			err = domain.ErrReservationExpired
			return
		}
		if current.State != domain.StateActive {
			switch current.State {
			case domain.StateExpired:
				err = domain.ErrReservationExpired
			default:
				err = domain.ErrReservationAlreadyFinalized
			}
			return
		}
		confirmedAt := now
		_ = s.repo.UpdateReservation(reservationID, func(r *domain.Reservation) error {
			r.State = domain.StateConfirmed
			r.ConfirmedAt = &confirmedAt
			result = *r
			return nil
		})
		_ = s.repo.UpdateProduct(current.ProductID, func(p *domain.ProductInventory) error {
			p.ActiveReservationCount--
			p.ConfirmedCount++
			return nil
		})
	})
	return result, err
}

// CancelReservation releases an active reservation's hold.
func (s *ReservationService) CancelReservation(_ context.Context, reservationID string) (domain.Reservation, error) {
	res, ok := s.repo.GetReservation(reservationID)
	if !ok {
		return domain.Reservation{}, domain.ErrReservationNotFound
	}
	var result domain.Reservation
	var err error
	s.locks.With(res.ProductID, func() {
		current, ok := s.repo.GetReservation(reservationID)
		if !ok {
			err = domain.ErrReservationNotFound
			return
		}
		if current.State != domain.StateActive {
			err = domain.ErrReservationAlreadyFinalized
			return
		}
		now := s.clock.Now()
		cancelledAt := now
		_ = s.repo.UpdateReservation(reservationID, func(r *domain.Reservation) error {
			r.State = domain.StateCancelled
			r.CancelledAt = &cancelledAt
			result = *r
			return nil
		})
		_ = s.repo.UpdateProduct(current.ProductID, func(p *domain.ProductInventory) error {
			p.ActiveReservationCount--
			return nil
		})
	})
	return result, err
}

// ExpireReservations transitions every active reservation whose ExpiresAt <= now to Expired
// and releases its inventory hold. Returns the number of reservations expired.
func (s *ReservationService) ExpireReservations(_ context.Context, now time.Time) int {
	byProduct := map[string][]string{}
	for _, r := range s.repo.AllActiveReservations() {
		if !now.Before(r.ExpiresAt) {
			byProduct[r.ProductID] = append(byProduct[r.ProductID], r.ReservationID)
		}
	}
	total := 0
	for productID, ids := range byProduct {
		s.locks.With(productID, func() {
			for _, id := range ids {
				if s.expireReservationLocked(id, now) {
					total++
				}
			}
		})
	}
	return total
}

// expireForProductLocked expires stale actives for one product.
// Caller MUST hold the per-product lock.
func (s *ReservationService) expireForProductLocked(productID string, now time.Time) {
	for _, r := range s.repo.ActiveReservationsByProduct(productID) {
		if !now.Before(r.ExpiresAt) {
			s.expireReservationLocked(r.ReservationID, now)
		}
	}
}

// expireReservationLocked expires a single reservation and releases its hold.
// Caller MUST hold the per-product lock for the reservation's product.
// Returns true if a transition happened.
func (s *ReservationService) expireReservationLocked(reservationID string, now time.Time) bool {
	current, ok := s.repo.GetReservation(reservationID)
	if !ok || current.State != domain.StateActive {
		return false
	}
	expiredAt := now
	_ = s.repo.UpdateReservation(reservationID, func(r *domain.Reservation) error {
		r.State = domain.StateExpired
		r.ExpiredAt = &expiredAt
		return nil
	})
	_ = s.repo.UpdateProduct(current.ProductID, func(p *domain.ProductInventory) error {
		p.ActiveReservationCount--
		return nil
	})
	return true
}

// GetAvailableStock returns the current available units for a product.
func (s *ReservationService) GetAvailableStock(_ context.Context, productID string) (int, error) {
	p, ok := s.repo.GetProduct(productID)
	if !ok {
		return 0, domain.ErrProductNotFound
	}
	return p.AvailableStock(), nil
}

// GetReservation returns a reservation by ID.
func (s *ReservationService) GetReservation(_ context.Context, reservationID string) (domain.Reservation, bool) {
	return s.repo.GetReservation(reservationID)
}
