package httpiface_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestHTTP_GetReservation_OK(t *testing.T) {
	srv, _ := newServer(t)
	defer srv.Close()
	crResp := postJSON(t, srv.URL+"/reservations", map[string]string{
		"productId": "p1", "userId": "alice",
	})
	var created struct {
		ReservationID string `json:"reservationId"`
	}
	_ = json.NewDecoder(crResp.Body).Decode(&created)

	resp, _ := http.Get(srv.URL + "/reservations/" + created.ReservationID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	var got struct {
		ReservationID string `json:"reservationId"`
		State         string `json:"state"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&got)
	if got.ReservationID != created.ReservationID {
		t.Errorf("want id=%s, got %s", created.ReservationID, got.ReservationID)
	}
	if got.State != "Active" {
		t.Errorf("want Active, got %s", got.State)
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

func TestHTTP_RejectsUnsupportedReservePayloadShape(t *testing.T) {
	srv, _ := newServer(t)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/reservations", "application/json", bytes.NewBufferString(`{"productId":"p1","userId":"alice","quantity":1}`))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("unknown field: want 400, got %d", resp.StatusCode)
	}

	resp, err = http.Post(srv.URL+"/reservations", "application/json", bytes.NewBufferString(`{"productId":"p1","userId":"alice"}{}`))
	if err != nil {
		t.Fatalf("post trailing json: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("trailing json: want 400, got %d", resp.StatusCode)
	}
}

func TestHTTP_RejectsInvalidIdentifiersBeforeApplicationService(t *testing.T) {
	srv, _ := newServer(t)
	defer srv.Close()

	resp := postJSON(t, srv.URL+"/reservations", map[string]string{"productId": "bad id", "userId": "alice"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("invalid reserve product id: want 400, got %d", resp.StatusCode)
	}

	resp = postJSON(t, srv.URL+"/reservations", map[string]string{"productId": "p1", "userId": "bad id"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("invalid reserve user id: want 400, got %d", resp.StatusCode)
	}

	resp, _ = http.Get(srv.URL + "/products/bad%20id/stock")
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("invalid stock product id: want 400, got %d", resp.StatusCode)
	}

	resp, _ = http.Post(srv.URL+"/reservations/bad%20id/confirm", "application/json", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("invalid confirm reservation id: want 400, got %d", resp.StatusCode)
	}
}

// compile-time assertion that the application service satisfies the interface.
var _ httpiface.ReservationService = (*application.ReservationService)(nil)
var _ context.Context = context.Background()

// Exercise each handler's error-path (404/409) so writeServiceError is fully covered.
func TestHTTP_ErrorPaths(t *testing.T) {
	srv, _ := newServer(t)
	defer srv.Close()

	// Unknown product stock -> 404.
	resp, _ := http.Get(srv.URL + "/products/ghost/stock")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("stock unknown: want 404, got %d", resp.StatusCode)
	}
	// Confirm unknown reservation -> 404.
	resp, _ = http.Post(srv.URL+"/reservations/nope/confirm", "application/json", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("confirm unknown: want 404, got %d", resp.StatusCode)
	}
	// Cancel unknown reservation -> 404.
	resp, _ = http.Post(srv.URL+"/reservations/nope/cancel", "application/json", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("cancel unknown: want 404, got %d", resp.StatusCode)
	}
	// Reserve unknown product -> 404.
	resp = postJSON(t, srv.URL+"/reservations", map[string]string{
		"productId": "ghost", "userId": "alice",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("reserve unknown product: want 404, got %d", resp.StatusCode)
	}

	// Create -> confirm once -> confirm again -> 409 (already finalized).
	crResp := postJSON(t, srv.URL+"/reservations", map[string]string{
		"productId": "p1", "userId": "alice",
	})
	var created struct {
		ReservationID string `json:"reservationId"`
	}
	_ = json.NewDecoder(crResp.Body).Decode(&created)
	confirmURL := srv.URL + "/reservations/" + created.ReservationID + "/confirm"
	_, _ = http.Post(confirmURL, "application/json", nil)
	resp, _ = http.Post(confirmURL, "application/json", nil)
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("double confirm: want 409, got %d", resp.StatusCode)
	}
}

// writeServiceError's default branch should map unknown errors to 500.
func TestHTTP_InternalError(t *testing.T) {
	h := httpiface.NewHandler(failingService{})
	req := httptest.NewRequest(http.MethodGet, "/products/p1/stock", nil)
	rr := httptest.NewRecorder()
	h.Routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("want 500, got %d", rr.Code)
	}
}

func TestHTTP_MetricsMiddlewareRecordsREDMetrics(t *testing.T) {
	repo := infrastructure.NewInMemoryRepository()
	repo.AddProduct(domain.ProductInventory{ProductID: "p1", TotalStock: 1})
	clk := infrastructure.NewFakeClock(time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC))
	svc := application.NewReservationService(repo, infrastructure.NewLockManager(), clk, 2*time.Minute)
	metrics := httpiface.NewMetrics()

	mux := http.NewServeMux()
	mux.Handle("/", metrics.Middleware(httpiface.NewHandler(svc).Routes()))
	mux.Handle("GET /metrics", metrics.Handler())
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/products/p1/stock")
	if err != nil {
		t.Fatalf("stock request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stock request: want 200, got %d", resp.StatusCode)
	}

	metricsResp, err := http.Get(srv.URL + "/metrics")
	if err != nil {
		t.Fatalf("metrics request: %v", err)
	}
	body, err := io.ReadAll(metricsResp.Body)
	if err != nil {
		t.Fatalf("read metrics: %v", err)
	}
	text := string(body)
	for _, want := range []string{
		`inventory_http_requests_total{method="GET",route="/products/{id}/stock",status="200"} 1`,
		`inventory_http_request_duration_seconds_bucket{method="GET",route="/products/{id}/stock",status="200",le="+Inf"} 1`,
		`inventory_http_request_duration_seconds_count{method="GET",route="/products/{id}/stock",status="200"} 1`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("metrics missing %q in:\n%s", want, text)
		}
	}
}

type failingService struct{}

func (failingService) ReserveItem(context.Context, string, string) (domain.Reservation, error) {
	return domain.Reservation{}, errFailure
}
func (failingService) ConfirmReservation(context.Context, string) (domain.Reservation, error) {
	return domain.Reservation{}, errFailure
}
func (failingService) CancelReservation(context.Context, string) (domain.Reservation, error) {
	return domain.Reservation{}, errFailure
}
func (failingService) GetAvailableStock(context.Context, string) (int, error) {
	return 0, errFailure
}
func (failingService) GetReservation(context.Context, string) (domain.Reservation, bool) {
	return domain.Reservation{}, false
}

var errFailure = errorString("unexpected")

type errorString string

func (e errorString) Error() string { return string(e) }
