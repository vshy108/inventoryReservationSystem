package application_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vshy108/inventoryReservationSystem/internal/application"
	"github.com/vshy108/inventoryReservationSystem/internal/domain"
	"github.com/vshy108/inventoryReservationSystem/internal/infrastructure"
)

func newSvc(t *testing.T, productID string, stock int) (*application.ReservationService, *infrastructure.FakeClock) {
	t.Helper()
	repo := infrastructure.NewInMemoryRepository()
	repo.AddProduct(domain.ProductInventory{ProductID: productID, TotalStock: stock})
	clk := infrastructure.NewFakeClock(time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC))
	svc := application.NewReservationService(repo, infrastructure.NewLockManager(), clk, 2*time.Minute)
	return svc, clk
}

func TestReserveItem_SucceedsWhenStockAvailable(t *testing.T) {
	svc, _ := newSvc(t, "p1", 1)
	res, err := svc.ReserveItem(context.Background(), "p1", "userA")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.State != domain.StateActive {
		t.Errorf("want Active, got %s", res.State)
	}
	avail, _ := svc.GetAvailableStock(context.Background(), "p1")
	if avail != 0 {
		t.Errorf("want available=0, got %d", avail)
	}
}

func TestReserveItem_FailsWhenStockUnavailable(t *testing.T) {
	svc, _ := newSvc(t, "p1", 1)
	if _, err := svc.ReserveItem(context.Background(), "p1", "userA"); err != nil {
		t.Fatalf("first reserve: %v", err)
	}
	_, err := svc.ReserveItem(context.Background(), "p1", "userB")
	if !errors.Is(err, domain.ErrOutOfStock) {
		t.Fatalf("want ErrOutOfStock, got %v", err)
	}
}

func TestReserveItem_UnknownProduct(t *testing.T) {
	svc, _ := newSvc(t, "p1", 1)
	_, err := svc.ReserveItem(context.Background(), "ghost", "userA")
	if !errors.Is(err, domain.ErrProductNotFound) {
		t.Fatalf("want ErrProductNotFound, got %v", err)
	}
}

func TestConfirmReservation_Finalizes(t *testing.T) {
	svc, _ := newSvc(t, "p1", 1)
	res, _ := svc.ReserveItem(context.Background(), "p1", "userA")
	confirmed, err := svc.ConfirmReservation(context.Background(), res.ReservationID)
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if confirmed.State != domain.StateConfirmed {
		t.Errorf("want Confirmed, got %s", confirmed.State)
	}
	avail, _ := svc.GetAvailableStock(context.Background(), "p1")
	if avail != 0 {
		t.Errorf("want available=0 after confirm, got %d", avail)
	}
}

func TestConfirmReservation_DoubleConfirm(t *testing.T) {
	svc, _ := newSvc(t, "p1", 1)
	res, _ := svc.ReserveItem(context.Background(), "p1", "userA")
	if _, err := svc.ConfirmReservation(context.Background(), res.ReservationID); err != nil {
		t.Fatalf("first confirm: %v", err)
	}
	_, err := svc.ConfirmReservation(context.Background(), res.ReservationID)
	if !errors.Is(err, domain.ErrReservationAlreadyFinalized) {
		t.Fatalf("want ErrReservationAlreadyFinalized, got %v", err)
	}
}

func TestCancelReservation_ReleasesStock(t *testing.T) {
	svc, _ := newSvc(t, "p1", 1)
	res, _ := svc.ReserveItem(context.Background(), "p1", "userA")
	if _, err := svc.CancelReservation(context.Background(), res.ReservationID); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	avail, _ := svc.GetAvailableStock(context.Background(), "p1")
	if avail != 1 {
		t.Errorf("want available=1 after cancel, got %d", avail)
	}
}

func TestExpireReservations_ReleasesExpiredStock(t *testing.T) {
	svc, clk := newSvc(t, "p1", 1)
	res, _ := svc.ReserveItem(context.Background(), "p1", "userA")
	clk.Advance(3 * time.Minute)
	n := svc.ExpireReservations(context.Background(), clk.Now())
	if n != 1 {
		t.Errorf("want 1 expired, got %d", n)
	}
	r, _ := svc.GetReservation(context.Background(), res.ReservationID)
	if r.State != domain.StateExpired {
		t.Errorf("want Expired, got %s", r.State)
	}
	avail, _ := svc.GetAvailableStock(context.Background(), "p1")
	if avail != 1 {
		t.Errorf("want available=1 after expire, got %d", avail)
	}
}

func TestConfirmAfterExpiry_Fails(t *testing.T) {
	svc, clk := newSvc(t, "p1", 1)
	res, _ := svc.ReserveItem(context.Background(), "p1", "userA")
	clk.Advance(3 * time.Minute)
	_, err := svc.ConfirmReservation(context.Background(), res.ReservationID)
	if !errors.Is(err, domain.ErrReservationExpired) {
		t.Fatalf("want ErrReservationExpired, got %v", err)
	}
}

// Level 3: 500 parallel reserve calls against stock=1 -> exactly 1 success.
func TestReserveItem_500Concurrent_Stock1(t *testing.T) {
	svc, _ := newSvc(t, "p1", 1)
	const N = 500
	var wg sync.WaitGroup
	var success, failure atomic.Int64
	start := make(chan struct{})
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			<-start
			_, err := svc.ReserveItem(context.Background(), "p1", "user")
			if err == nil {
				success.Add(1)
			} else if errors.Is(err, domain.ErrOutOfStock) {
				failure.Add(1)
			} else {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()
	if success.Load() != 1 {
		t.Errorf("want 1 success, got %d", success.Load())
	}
	if failure.Load() != N-1 {
		t.Errorf("want %d failures, got %d", N-1, failure.Load())
	}
	avail, _ := svc.GetAvailableStock(context.Background(), "p1")
	if avail != 0 {
		t.Errorf("want available=0, got %d", avail)
	}
}
