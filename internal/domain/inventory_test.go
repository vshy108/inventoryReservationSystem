package domain_test

import (
	"testing"

	"everest/inventoryReservation/internal/domain"
)

// SF-S1/S2/S3: AvailableStock = TotalStock - ConfirmedCount - ActiveReservationCount.
func TestAvailableStock_Formula(t *testing.T) {
	cases := []struct {
		name string
		p    domain.ProductInventory
		want int
	}{
		{"SF-S1", domain.ProductInventory{TotalStock: 5, ConfirmedCount: 1, ActiveReservationCount: 2}, 2},
		{"SF-S2", domain.ProductInventory{TotalStock: 1, ConfirmedCount: 0, ActiveReservationCount: 1}, 0},
		{"SF-S3", domain.ProductInventory{TotalStock: 1, ConfirmedCount: 1, ActiveReservationCount: 0}, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.p.AvailableStock(); got != c.want {
				t.Errorf("want %d, got %d", c.want, got)
			}
		})
	}
}
