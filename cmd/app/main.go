package main

import (
	"context"
	"fmt"
	"time"

	"github.com/vshy108/inventoryReservationSystem/internal/application"
	"github.com/vshy108/inventoryReservationSystem/internal/domain"
	"github.com/vshy108/inventoryReservationSystem/internal/infrastructure"
)

func main() {
	repo := infrastructure.NewInMemoryRepository()
	repo.AddProduct(domain.ProductInventory{ProductID: "sku-1", TotalStock: 1})
	svc := application.NewReservationService(
		repo,
		infrastructure.NewLockManager(),
		infrastructure.SystemClock{},
		2*time.Minute,
	)
	ctx := context.Background()

	r1, err := svc.ReserveItem(ctx, "sku-1", "alice")
	fmt.Printf("alice reserve: id=%s state=%s err=%v\n", r1.ReservationID, r1.State, err)

	_, err = svc.ReserveItem(ctx, "sku-1", "bob")
	fmt.Printf("bob reserve err=%v (expected: out of stock)\n", err)

	r1c, err := svc.ConfirmReservation(ctx, r1.ReservationID)
	fmt.Printf("confirm alice: state=%s err=%v\n", r1c.State, err)

	avail, _ := svc.GetAvailableStock(ctx, "sku-1")
	fmt.Printf("sku-1 available=%d (expected: 0)\n", avail)
}
