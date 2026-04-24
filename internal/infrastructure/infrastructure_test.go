package infrastructure_test

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"everest/inventoryReservation/internal/domain"
	"everest/inventoryReservation/internal/infrastructure"
)

func TestSystemClock_NowAdvances(t *testing.T) {
	var c infrastructure.SystemClock
	a := c.Now()
	b := c.Now()
	if b.Before(a) {
		t.Fatalf("SystemClock.Now went backwards: a=%v b=%v", a, b)
	}
	if a.IsZero() || b.IsZero() {
		t.Fatalf("SystemClock.Now returned zero time")
	}
}

func TestFakeClock_Advance(t *testing.T) {
	start := time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)
	c := infrastructure.NewFakeClock(start)
	if !c.Now().Equal(start) {
		t.Fatalf("want %v, got %v", start, c.Now())
	}
	c.Advance(2 * time.Minute)
	if !c.Now().Equal(start.Add(2 * time.Minute)) {
		t.Fatalf("advance failed: %v", c.Now())
	}
}

func TestLockManager_SerialisesCriticalSection(t *testing.T) {
	lm := infrastructure.NewLockManager()
	var inCritical atomic.Int32
	var violations atomic.Int32
	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)
	start := make(chan struct{})
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			<-start
			lm.With("k", func() {
				if inCritical.Add(1) > 1 {
					violations.Add(1)
				}
				time.Sleep(time.Microsecond)
				inCritical.Add(-1)
			})
		}()
	}
	close(start)
	wg.Wait()
	if violations.Load() != 0 {
		t.Fatalf("lock manager allowed %d concurrent critical sections", violations.Load())
	}
}

func TestLockManager_DifferentKeysIndependent(t *testing.T) {
	lm := infrastructure.NewLockManager()
	called := make(chan struct{}, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	// Hold k1 for a bit, meanwhile k2 must be able to proceed.
	release := make(chan struct{})
	go func() {
		defer wg.Done()
		lm.With("k1", func() {
			called <- struct{}{}
			<-release
		})
	}()
	<-called
	go func() {
		defer wg.Done()
		lm.With("k2", func() { called <- struct{}{} })
	}()
	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("k2 blocked by k1; keys are not independent")
	}
	close(release)
	wg.Wait()
}

func TestInMemoryRepository_UnknownKeys(t *testing.T) {
	r := infrastructure.NewInMemoryRepository()

	if _, ok := r.GetProduct("missing"); ok {
		t.Error("GetProduct returned ok for missing product")
	}
	if _, ok := r.GetReservation("missing"); ok {
		t.Error("GetReservation returned ok for missing reservation")
	}
	if err := r.UpdateProduct("missing", func(*domain.ProductInventory) error { return nil }); !errors.Is(err, domain.ErrProductNotFound) {
		t.Errorf("UpdateProduct missing: want ErrProductNotFound, got %v", err)
	}
	if err := r.UpdateReservation("missing", func(*domain.Reservation) error { return nil }); !errors.Is(err, domain.ErrReservationNotFound) {
		t.Errorf("UpdateReservation missing: want ErrReservationNotFound, got %v", err)
	}
	if got := r.ActiveReservationsByProduct("missing"); len(got) != 0 {
		t.Errorf("want empty, got %v", got)
	}
	if got := r.AllActiveReservations(); len(got) != 0 {
		t.Errorf("want empty, got %v", got)
	}
	if got := r.AllEvents(); len(got) != 0 {
		t.Errorf("want empty, got %v", got)
	}
}

func TestInMemoryRepository_UpdateReturnsCallbackError(t *testing.T) {
	r := infrastructure.NewInMemoryRepository()
	r.AddProduct(domain.ProductInventory{ProductID: "p1", TotalStock: 1})
	boom := errors.New("boom")
	if err := r.UpdateProduct("p1", func(*domain.ProductInventory) error { return boom }); err != boom {
		t.Errorf("UpdateProduct: want boom, got %v", err)
	}
	r.SaveReservation(domain.Reservation{ReservationID: "r1", ProductID: "p1", State: domain.StateActive})
	if err := r.UpdateReservation("r1", func(*domain.Reservation) error { return boom }); err != boom {
		t.Errorf("UpdateReservation: want boom, got %v", err)
	}
}
