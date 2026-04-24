package domain

// ProductInventory holds per-product stock counters.
//
// Available stock formula:
//
//	Available = TotalStock - ConfirmedCount - ActiveReservationCount
type ProductInventory struct {
	ProductID              string
	TotalStock             int
	ConfirmedCount         int
	ActiveReservationCount int
}

// AvailableStock returns units currently reservable.
func (p ProductInventory) AvailableStock() int {
	return p.TotalStock - p.ConfirmedCount - p.ActiveReservationCount
}
