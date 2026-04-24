package infrastructure

import (
	"sync"

	"github.com/vshy108/inventoryReservationSystem/internal/domain"
)

// InMemoryRepository stores products and reservations in memory.
// Its own mutex only protects the internal maps; business invariants
// (such as "available stock never goes negative") are enforced by the
// service layer using the per-product LockManager.
type InMemoryRepository struct {
	mu           sync.RWMutex
	products     map[string]*domain.ProductInventory
	reservations map[string]*domain.Reservation
}

// NewInMemoryRepository constructs an empty repository.
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		products:     make(map[string]*domain.ProductInventory),
		reservations: make(map[string]*domain.Reservation),
	}
}

// AddProduct inserts or replaces a product inventory record.
func (r *InMemoryRepository) AddProduct(p domain.ProductInventory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := p
	r.products[p.ProductID] = &cp
}

// GetProduct returns a copy of the product inventory.
func (r *InMemoryRepository) GetProduct(id string) (domain.ProductInventory, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.products[id]
	if !ok {
		return domain.ProductInventory{}, false
	}
	return *p, true
}

// UpdateProduct mutates a product under the repository's write lock.
// The caller is expected to hold the per-product lock so that business
// invariants remain consistent across multiple repo calls.
func (r *InMemoryRepository) UpdateProduct(id string, fn func(*domain.ProductInventory) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.products[id]
	if !ok {
		return domain.ErrProductNotFound
	}
	return fn(p)
}

// SaveReservation inserts or replaces a reservation.
func (r *InMemoryRepository) SaveReservation(res domain.Reservation) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := res
	r.reservations[res.ReservationID] = &cp
}

// GetReservation returns a copy of the reservation.
func (r *InMemoryRepository) GetReservation(id string) (domain.Reservation, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res, ok := r.reservations[id]
	if !ok {
		return domain.Reservation{}, false
	}
	return *res, true
}

// UpdateReservation mutates a reservation under the repository's write lock.
func (r *InMemoryRepository) UpdateReservation(id string, fn func(*domain.Reservation) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	res, ok := r.reservations[id]
	if !ok {
		return domain.ErrReservationNotFound
	}
	return fn(res)
}

// ActiveReservationsByProduct returns copies of active reservations for a product.
func (r *InMemoryRepository) ActiveReservationsByProduct(productID string) []domain.Reservation {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.Reservation, 0)
	for _, res := range r.reservations {
		if res.ProductID == productID && res.State == domain.StateActive {
			out = append(out, *res)
		}
	}
	return out
}

// AllActiveReservations returns copies of every currently active reservation.
func (r *InMemoryRepository) AllActiveReservations() []domain.Reservation {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.Reservation, 0)
	for _, res := range r.reservations {
		if res.State == domain.StateActive {
			out = append(out, *res)
		}
	}
	return out
}
