package httpiface_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"everest/inventoryReservation/internal/application"
	"everest/inventoryReservation/internal/domain"
	"everest/inventoryReservation/internal/infrastructure"
	httpiface "everest/inventoryReservation/internal/interface/http"
)

func newServer(t *testing.T) (*httptest.Server, *infrastructure.FakeClock) {
	t.Helper()
	repo := infrastructure.NewInMemoryRepository()
	repo.AddProduct(domain.ProductInventory{ProductID: "p1", TotalStock: 1})
	clk := infrastructure.NewFakeClock(time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC))
	svc := application.NewReservationService(repo, infrastructure.NewLockManager(), clk, 2*time.Minute)
	mux := httpiface.NewHandler(svc).Routes()
	return httptest.NewServer(mux), clk
}

func postJSON(t *testing.T, url string, body any) *http.Response {
	t.Helper()
	buf := new(bytes.Buffer)
	if body != nil {
		if err := json.NewEncoder(buf).Encode(body); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}
	resp, err := http.Post(url, "application/json", buf)
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	return resp
}

func TestHTTP_ReserveConfirmStock(t *testing.T) {
	srv, _ := newServer(t)
	defer srv.Close()

	// Reserve
	resp := postJSON(t, srv.URL+"/reservations", map[string]string{
		"productId": "p1", "userId": "alice",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("want 201, got %d", resp.StatusCode)
	}
	var created struct {
		ReservationID string `json:"reservationId"`
		State         string `json:"state"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if created.State != string(domain.StateActive) {
		t.Errorf("want Active, got %s", created.State)
	}

	// Stock should be 0
	sResp, _ := http.Get(srv.URL + "/products/p1/stock")
	if sResp.StatusCode != 200 {
		t.Fatalf("want 200, got %d", sResp.StatusCode)
	}
	var stock struct {
		Available int `json:"available"`
	}
	_ = json.NewDecoder(sResp.Body).Decode(&stock)
	if stock.Available != 0 {
		t.Errorf("want available=0, got %d", stock.Available)
	}

	// Second reserve -> 409 out of stock
	resp2 := postJSON(t, srv.URL+"/reservations", map[string]string{
		"productId": "p1", "userId": "bob",
	})
	if resp2.StatusCode != http.StatusConflict {
		t.Errorf("want 409, got %d", resp2.StatusCode)
	}

	// Confirm
	confirmURL := srv.URL + "/reservations/" + created.ReservationID + "/confirm"
	cResp, err := http.Post(confirmURL, "application/json", nil)
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if cResp.StatusCode != 200 {
		t.Errorf("want 200 on confirm, got %d", cResp.StatusCode)
	}
}

func TestHTTP_CancelReleasesStock(t *testing.T) {
	srv, _ := newServer(t)
	defer srv.Close()

	resp := postJSON(t, srv.URL+"/reservations", map[string]string{
		"productId": "p1", "userId": "alice",
	})
	var created struct {
		ReservationID string `json:"reservationId"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&created)

	cancelURL := srv.URL + "/reservations/" + created.ReservationID + "/cancel"
	cResp, err := http.Post(cancelURL, "application/json", nil)
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if cResp.StatusCode != 200 {
		t.Errorf("want 200 on cancel, got %d", cResp.StatusCode)
	}

	sResp, _ := http.Get(srv.URL + "/products/p1/stock")
	var stock struct {
		Available int `json:"available"`
	}
	_ = json.NewDecoder(sResp.Body).Decode(&stock)
	if stock.Available != 1 {
		t.Errorf("want available=1 after cancel, got %d", stock.Available)
	}
}

func TestHTTP_UnknownReservation(t *testing.T) {
	srv, _ := newServer(t)
	defer srv.Close()
	resp, _ := http.Get(srv.URL + "/reservations/does-not-exist")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("want 404, got %d", resp.StatusCode)
	}
}

func TestHTTP_BadReserveBody(t *testing.T) {
	srv, _ := newServer(t)
	defer srv.Close()
	resp, err := http.Post(srv.URL+"/reservations", "application/json", bytes.NewBufferString("not-json"))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("want 400, got %d", resp.StatusCode)
	}

	resp2 := postJSON(t, srv.URL+"/reservations", map[string]string{"productId": "", "userId": ""})
	if resp2.StatusCode != http.StatusBadRequest {
		t.Errorf("want 400 on missing fields, got %d", resp2.StatusCode)
	}
}

// compile-time assertion that the application service satisfies the interface.
var _ httpiface.ReservationService = (*application.ReservationService)(nil)
var _ context.Context = context.Background()
