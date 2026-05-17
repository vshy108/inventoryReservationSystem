package httpiface

import (
	_ "embed"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"everest/inventoryReservation/internal/domain"
)

//go:embed openapi.yaml
var openAPISpec []byte

// ReservationService is the subset of the application service used by the
// HTTP handlers. Declaring it here keeps the interface layer decoupled
// from the concrete service implementation and easy to mock in tests.
type ReservationService interface {
	ReserveItem(ctx context.Context, productID, userID string) (domain.Reservation, error)
	ConfirmReservation(ctx context.Context, reservationID string) (domain.Reservation, error)
	CancelReservation(ctx context.Context, reservationID string) (domain.Reservation, error)
	GetAvailableStock(ctx context.Context, productID string) (int, error)
	GetReservation(ctx context.Context, reservationID string) (domain.Reservation, bool)
}

// Handler is the HTTP adapter for ReservationService.
type Handler struct {
	svc ReservationService
}

// NewHandler constructs a Handler.
func NewHandler(svc ReservationService) *Handler { return &Handler{svc: svc} }

// Routes returns an *http.ServeMux configured with all reservation endpoints.
//
// Endpoints:
//
//	POST /reservations                     -> reserve item
//	POST /reservations/{id}/confirm        -> confirm
//	POST /reservations/{id}/cancel         -> cancel
//	GET  /reservations/{id}                -> fetch
//	GET  /products/{id}/stock              -> available stock
//	GET  /openapi.yaml                     -> OpenAPI 3.0 spec (this package)
func (h *Handler) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /reservations", h.reserve)
	mux.HandleFunc("POST /reservations/{id}/confirm", h.confirm)
	mux.HandleFunc("POST /reservations/{id}/cancel", h.cancel)
	mux.HandleFunc("GET /reservations/{id}", h.getReservation)
	mux.HandleFunc("GET /products/{id}/stock", h.stock)
	mux.HandleFunc("GET /openapi.yaml", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		_, _ = w.Write(openAPISpec)
	})
	return mux
}

type reserveRequest struct {
	ProductID string `json:"productId"`
	UserID    string `json:"userId"`
}

type reservationResponse struct {
	ReservationID string    `json:"reservationId"`
	ProductID     string    `json:"productId"`
	UserID        string    `json:"userId"`
	State         string    `json:"state"`
	CreatedAt     time.Time `json:"createdAt"`
	ExpiresAt     time.Time `json:"expiresAt"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type stockResponse struct {
	ProductID string `json:"productId"`
	Available int    `json:"available"`
}

func (h *Handler) reserve(w http.ResponseWriter, r *http.Request) {
	var req reserveRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	// FIX: The old decoder accepted trailing JSON values and unknown fields, so
	// malformed request shapes could reach the application layer. A second decode
	// must hit EOF, and DisallowUnknownFields keeps the HTTP contract closed.
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if strings.TrimSpace(req.ProductID) == "" || strings.TrimSpace(req.UserID) == "" {
		writeError(w, http.StatusBadRequest, "productId and userId are required")
		return
	}
	if !validExternalID(req.ProductID) || !validExternalID(req.UserID) {
		writeError(w, http.StatusBadRequest, "productId and userId must use letters, digits, '-' or '_'")
		return
	}
	res, err := h.svc.ReserveItem(r.Context(), req.ProductID, req.UserID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toResponse(res))
}

func (h *Handler) confirm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validExternalID(id) {
		writeError(w, http.StatusBadRequest, "reservation id is invalid")
		return
	}
	res, err := h.svc.ConfirmReservation(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toResponse(res))
}

func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validExternalID(id) {
		writeError(w, http.StatusBadRequest, "reservation id is invalid")
		return
	}
	res, err := h.svc.CancelReservation(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toResponse(res))
}

func (h *Handler) getReservation(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validExternalID(id) {
		writeError(w, http.StatusBadRequest, "reservation id is invalid")
		return
	}
	res, ok := h.svc.GetReservation(r.Context(), id)
	if !ok {
		writeError(w, http.StatusNotFound, domain.ErrReservationNotFound.Error())
		return
	}
	writeJSON(w, http.StatusOK, toResponse(res))
}

func (h *Handler) stock(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validExternalID(id) {
		writeError(w, http.StatusBadRequest, "product id is invalid")
		return
	}
	avail, err := h.svc.GetAvailableStock(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stockResponse{ProductID: id, Available: avail})
}

func toResponse(r domain.Reservation) reservationResponse {
	return reservationResponse{
		ReservationID: r.ReservationID,
		ProductID:     r.ProductID,
		UserID:        r.UserID,
		State:         string(r.State),
		CreatedAt:     r.CreatedAt,
		ExpiresAt:     r.ExpiresAt,
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrProductNotFound),
		errors.Is(err, domain.ErrReservationNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrOutOfStock):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrReservationAlreadyFinalized),
		errors.Is(err, domain.ErrReservationExpired):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

func validExternalID(id string) bool {
	if id == "" || len(id) > 64 || strings.TrimSpace(id) != id {
		return false
	}
	// FIX: Previously any non-empty path/body string crossed the HTTP boundary,
	// which blurred malformed input with valid unknown resources. Restricting IDs
	// here keeps bad syntax as 400 while valid missing resources still return 404.
	for _, char := range id {
		if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || char == '-' || char == '_' {
			continue
		}
		return false
	}
	return true
}
