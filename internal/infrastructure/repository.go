package infrastructure

import "everest/inventoryReservation/internal/domain"

// Repository is the persistence contract used by the service layer.
// Both InMemoryRepository and PostgresRepository satisfy this interface.
type Repository interface {
	AppendEvent(e domain.InventoryEvent)
	AllEvents() []domain.InventoryEvent

	AddProduct(p domain.ProductInventory)
	GetProduct(id string) (domain.ProductInventory, bool)
	UpdateProduct(id string, fn func(*domain.ProductInventory) error) error

	SaveReservation(res domain.Reservation)
	GetReservation(id string) (domain.Reservation, bool)
	UpdateReservation(id string, fn func(*domain.Reservation) error) error

	ActiveReservationsByProduct(productID string) []domain.Reservation
	AllActiveReservations() []domain.Reservation
}
