package application_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"everest/inventoryReservation/internal/application"
	"everest/inventoryReservation/internal/domain"
	"everest/inventoryReservation/internal/infrastructure"
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

func TestReserveAfterExpiry_FreesSlot(t *testing.T) {
	svc, clk := newSvc(t, "p1", 1)
	first, _ := svc.ReserveItem(context.Background(), "p1", "userA")
	clk.Advance(3 * time.Minute)
	// Lazy expiry inside ReserveItem should release the slot for userB.
	second, err := svc.ReserveItem(context.Background(), "p1", "userB")
	if err != nil {
		t.Fatalf("second reserve after expiry: %v", err)
	}
	if second.ReservationID == first.ReservationID {
		t.Fatalf("expected new reservation id, got same %s", first.ReservationID)
	}
	r1, _ := svc.GetReservation(context.Background(), first.ReservationID)
	if r1.State != domain.StateExpired {
		t.Errorf("want first Expired, got %s", r1.State)
	}
	avail, _ := svc.GetAvailableStock(context.Background(), "p1")
	if avail != 0 {
		t.Errorf("want available=0, got %d", avail)
	}
}

func TestCancelThenReReserve_Succeeds(t *testing.T) {
	svc, _ := newSvc(t, "p1", 1)
	first, _ := svc.ReserveItem(context.Background(), "p1", "userA")
	if _, err := svc.CancelReservation(context.Background(), first.ReservationID); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	second, err := svc.ReserveItem(context.Background(), "p1", "userB")
	if err != nil {
		t.Fatalf("reserve after cancel: %v", err)
	}
	if second.State != domain.StateActive {
		t.Errorf("want Active, got %s", second.State)
	}
	// Cancelling an already-cancelled reservation is a finalized-state error.
	_, err = svc.CancelReservation(context.Background(), first.ReservationID)
	if !errors.Is(err, domain.ErrReservationAlreadyFinalized) {
		t.Fatalf("want ErrReservationAlreadyFinalized, got %v", err)
	}
}

func TestMultiProduct_Isolation(t *testing.T) {
	repo := infrastructure.NewInMemoryRepository()
	repo.AddProduct(domain.ProductInventory{ProductID: "p1", TotalStock: 1})
	repo.AddProduct(domain.ProductInventory{ProductID: "p2", TotalStock: 1})
	clk := infrastructure.NewFakeClock(time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC))
	svc := application.NewReservationService(repo, infrastructure.NewLockManager(), clk, 2*time.Minute)

	if _, err := svc.ReserveItem(context.Background(), "p1", "userA"); err != nil {
		t.Fatalf("reserve p1: %v", err)
	}
	// p1 is full, but p2 should remain independent.
	if _, err := svc.ReserveItem(context.Background(), "p1", "userB"); !errors.Is(err, domain.ErrOutOfStock) {
		t.Fatalf("want ErrOutOfStock on p1, got %v", err)
	}
	if _, err := svc.ReserveItem(context.Background(), "p2", "userB"); err != nil {
		t.Fatalf("reserve p2 should succeed: %v", err)
	}
	a1, _ := svc.GetAvailableStock(context.Background(), "p1")
	a2, _ := svc.GetAvailableStock(context.Background(), "p2")
	if a1 != 0 || a2 != 0 {
		t.Errorf("want p1=0 p2=0, got p1=%d p2=%d", a1, a2)
	}
}

func TestInventoryEventLog_RecordsTransitions(t *testing.T) {
	svc, clk := newSvc(t, "p1", 2)
	r1, _ := svc.ReserveItem(context.Background(), "p1", "userA")
	r2, _ := svc.ReserveItem(context.Background(), "p1", "userB")
	if _, err := svc.ConfirmReservation(context.Background(), r1.ReservationID); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if _, err := svc.CancelReservation(context.Background(), r2.ReservationID); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	r3, _ := svc.ReserveItem(context.Background(), "p1", "userC")
	clk.Advance(3 * time.Minute)
	if n := svc.ExpireReservations(context.Background(), clk.Now()); n != 1 {
		t.Fatalf("want 1 expired, got %d", n)
	}

	events := svc.Events()
	want := []struct {
		typ domain.InventoryEventType
		id  string
	}{
		{domain.EventReserved, r1.ReservationID},
		{domain.EventReserved, r2.ReservationID},
		{domain.EventConfirmed, r1.ReservationID},
		{domain.EventCancelled, r2.ReservationID},
		{domain.EventReserved, r3.ReservationID},
		{domain.EventExpired, r3.ReservationID},
	}
	if len(events) != len(want) {
		t.Fatalf("want %d events, got %d: %+v", len(want), len(events), events)
	}
	for i, w := range want {
		if events[i].Type != w.typ || events[i].ReservationID == nil || *events[i].ReservationID != w.id {
			gotID := "<nil>"
			if events[i].ReservationID != nil {
				gotID = *events[i].ReservationID
			}
			t.Errorf("event[%d]: want {%s %s}, got {%s %s}", i, w.typ, w.id, events[i].Type, gotID)
		}
		if events[i].EventID == "" {
			t.Errorf("event[%d]: EventID must be set", i)
		}
	}
}

func TestShadowRecorder_RecordsSuccessfulTransitions(t *testing.T) {
	repo := infrastructure.NewInMemoryRepository()
	repo.AddProduct(domain.ProductInventory{ProductID: "p1", TotalStock: 2})
	clk := infrastructure.NewFakeClock(time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC))
	recorder := &recordingShadowRecorder{}
	svc := application.NewReservationServiceWithShadowRecorder(repo, infrastructure.NewLockManager(), clk, 2*time.Minute, recorder)

	first, err := svc.ReserveItem(context.Background(), "p1", "userA")
	if err != nil {
		t.Fatalf("reserve first: %v", err)
	}
	second, err := svc.ReserveItem(context.Background(), "p1", "userB")
	if err != nil {
		t.Fatalf("reserve second: %v", err)
	}
	if _, err := svc.ConfirmReservation(context.Background(), first.ReservationID); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if _, err := svc.CancelReservation(context.Background(), second.ReservationID); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	third, err := svc.ReserveItem(context.Background(), "p1", "userC")
	if err != nil {
		t.Fatalf("reserve third: %v", err)
	}
	clk.Advance(3 * time.Minute)
	if expired := svc.ExpireReservations(context.Background(), clk.Now()); expired != 1 {
		t.Fatalf("want 1 expired, got %d", expired)
	}

	records := recorder.Records()
	want := []struct {
		action        application.ShadowAction
		reservationID string
		eventType     domain.InventoryEventType
	}{
		{application.ShadowReserved, first.ReservationID, domain.EventReserved},
		{application.ShadowReserved, second.ReservationID, domain.EventReserved},
		{application.ShadowConfirmed, first.ReservationID, domain.EventConfirmed},
		{application.ShadowCancelled, second.ReservationID, domain.EventCancelled},
		{application.ShadowReserved, third.ReservationID, domain.EventReserved},
		{application.ShadowExpired, third.ReservationID, domain.EventExpired},
	}
	if len(records) != len(want) {
		t.Fatalf("want %d shadow records, got %d", len(want), len(records))
	}
	for index, expected := range want {
		got := records[index]
		if got.Action != expected.action || got.Reservation.ReservationID != expected.reservationID || got.Event.Type != expected.eventType {
			t.Fatalf("record %d = (%s, %s, %s), want (%s, %s, %s)", index, got.Action, got.Reservation.ReservationID, got.Event.Type, expected.action, expected.reservationID, expected.eventType)
		}
		if got.Event.EventID == "" {
			t.Fatalf("record %d missing event id", index)
		}
	}
}

type recordingShadowRecorder struct {
	mu      sync.Mutex
	records []application.ShadowRecord
}

func (recorder *recordingShadowRecorder) RecordShadow(_ context.Context, record application.ShadowRecord) {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	recorder.records = append(recorder.records, record)
}

func (recorder *recordingShadowRecorder) Records() []application.ShadowRecord {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	records := make([]application.ShadowRecord, len(recorder.records))
	copy(records, recorder.records)
	return records
}

// EX-R4: ExpireReservations is idempotent; running twice at the same now
// returns 0 the second time and does not double-release stock.
func TestExpireReservations_Idempotent(t *testing.T) {
	svc, clk := newSvc(t, "p1", 1)
	if _, err := svc.ReserveItem(context.Background(), "p1", "userA"); err != nil {
		t.Fatalf("reserve: %v", err)
	}
	clk.Advance(3 * time.Minute)
	if n := svc.ExpireReservations(context.Background(), clk.Now()); n != 1 {
		t.Fatalf("first sweep: want 1, got %d", n)
	}
	if n := svc.ExpireReservations(context.Background(), clk.Now()); n != 0 {
		t.Fatalf("second sweep: want 0, got %d", n)
	}
	avail, _ := svc.GetAvailableStock(context.Background(), "p1")
	if avail != 1 {
		t.Errorf("want available=1 after idempotent sweep, got %d", avail)
	}
}

// CC-R6: Cancelling a non-Active reservation returns ErrReservationAlreadyFinalized.
func TestCancel_Confirmed_Fails(t *testing.T) {
	svc, _ := newSvc(t, "p1", 1)
	res, _ := svc.ReserveItem(context.Background(), "p1", "userA")
	if _, err := svc.ConfirmReservation(context.Background(), res.ReservationID); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	_, err := svc.CancelReservation(context.Background(), res.ReservationID)
	if !errors.Is(err, domain.ErrReservationAlreadyFinalized) {
		t.Fatalf("want ErrReservationAlreadyFinalized, got %v", err)
	}
}

// CC/RL unknown id branches.
func TestConfirmCancel_UnknownID(t *testing.T) {
	svc, _ := newSvc(t, "p1", 1)
	if _, err := svc.ConfirmReservation(context.Background(), "nope"); !errors.Is(err, domain.ErrReservationNotFound) {
		t.Errorf("confirm unknown: want ErrReservationNotFound, got %v", err)
	}
	if _, err := svc.CancelReservation(context.Background(), "nope"); !errors.Is(err, domain.ErrReservationNotFound) {
		t.Errorf("cancel unknown: want ErrReservationNotFound, got %v", err)
	}
}

// A ReserveRejected event is recorded when a reservation fails due to ErrOutOfStock.
func TestReserveRejected_EmitsEvent(t *testing.T) {
	svc, _ := newSvc(t, "p1", 1)
	if _, err := svc.ReserveItem(context.Background(), "p1", "userA"); err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if _, err := svc.ReserveItem(context.Background(), "p1", "userB"); !errors.Is(err, domain.ErrOutOfStock) {
		t.Fatalf("want ErrOutOfStock, got %v", err)
	}
	events := svc.Events()
	var rejected *domain.InventoryEvent
	for i := range events {
		if events[i].Type == domain.EventReserveRejected {
			rejected = &events[i]
			break
		}
	}
	if rejected == nil {
		t.Fatalf("expected a ReserveRejected event, got %+v", events)
	}
	if rejected.UserID == nil || *rejected.UserID != "userB" {
		t.Errorf("want UserID=userB, got %v", rejected.UserID)
	}
	if rejected.Reason == nil || *rejected.Reason == "" {
		t.Errorf("want a non-empty Reason, got %v", rejected.Reason)
	}
	if rejected.ReservationID != nil {
		t.Errorf("ReserveRejected should not reference a reservation, got %v", *rejected.ReservationID)
	}
}

// Determinism under repeated high-contention runs: every run with stock=N
// and K>N concurrent requests yields exactly N successes.
func TestReserveItem_RepeatedConcurrencyDeterminism(t *testing.T) {
	const runs = 10
	const N = 3
	const K = 200
	for run := 0; run < runs; run++ {
		svc, _ := newSvc(t, "p1", N)
		var wg sync.WaitGroup
		var success, failure atomic.Int64
		start := make(chan struct{})
		wg.Add(K)
		for i := 0; i < K; i++ {
			go func() {
				defer wg.Done()
				<-start
				_, err := svc.ReserveItem(context.Background(), "p1", "user")
				if err == nil {
					success.Add(1)
				} else if errors.Is(err, domain.ErrOutOfStock) {
					failure.Add(1)
				} else {
					t.Errorf("run %d: unexpected error: %v", run, err)
				}
			}()
		}
		close(start)
		wg.Wait()
		if success.Load() != int64(N) || failure.Load() != int64(K-N) {
			t.Fatalf("run %d: want %d success %d failure, got %d/%d",
				run, N, K-N, success.Load(), failure.Load())
		}
		avail, _ := svc.GetAvailableStock(context.Background(), "p1")
		if avail != 0 {
			t.Fatalf("run %d: want available=0, got %d", run, avail)
		}
	}
}

// NewReservationService with holdFor=0 falls back to DefaultHoldDuration.
func TestNewReservationService_DefaultHold(t *testing.T) {
	repo := infrastructure.NewInMemoryRepository()
	repo.AddProduct(domain.ProductInventory{ProductID: "p1", TotalStock: 1})
	clk := infrastructure.NewFakeClock(time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC))
	svc := application.NewReservationService(repo, infrastructure.NewLockManager(), clk, 0)
	res, err := svc.ReserveItem(context.Background(), "p1", "userA")
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if got := res.ExpiresAt.Sub(res.CreatedAt); got != application.DefaultHoldDuration {
		t.Errorf("want hold=%s, got %s", application.DefaultHoldDuration, got)
	}
}

// GetAvailableStock returns ErrProductNotFound for unknown products.
func TestGetAvailableStock_UnknownProduct(t *testing.T) {
	svc, _ := newSvc(t, "p1", 1)
	if _, err := svc.GetAvailableStock(context.Background(), "ghost"); !errors.Is(err, domain.ErrProductNotFound) {
		t.Errorf("want ErrProductNotFound, got %v", err)
	}
}

// Confirming an already-Expired (not just past-expiry-but-Active) reservation
// returns ErrReservationExpired.
func TestConfirm_AlreadyExpired(t *testing.T) {
	svc, clk := newSvc(t, "p1", 1)
	res, _ := svc.ReserveItem(context.Background(), "p1", "userA")
	clk.Advance(3 * time.Minute)
	if n := svc.ExpireReservations(context.Background(), clk.Now()); n != 1 {
		t.Fatalf("want 1 expired, got %d", n)
	}
	_, err := svc.ConfirmReservation(context.Background(), res.ReservationID)
	if !errors.Is(err, domain.ErrReservationExpired) {
		t.Errorf("want ErrReservationExpired, got %v", err)
	}
}

// GetReservation returns ok=false for unknown IDs.
func TestGetReservation_Unknown(t *testing.T) {
	svc, _ := newSvc(t, "p1", 1)
	if _, ok := svc.GetReservation(context.Background(), "nope"); ok {
		t.Error("want ok=false for unknown reservation")
	}
}
