package httpiface

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"everest/inventoryReservation/internal/domain"
)

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
func (h *Handler) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /reservations", h.reserve)
	mux.HandleFunc("POST /reservations/{id}/confirm", h.confirm)
	mux.HandleFunc("POST /reservations/{id}/cancel", h.cancel)
	mux.HandleFunc("GET /reservations/{id}", h.getReservation)
	mux.HandleFunc("GET /products/{id}/stock", h.stock)
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if strings.TrimSpace(req.ProductID) == "" || strings.TrimSpace(req.UserID) == "" {
		writeError(w, http.StatusBadRequest, "productId and userId are required")
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
	res, err := h.svc.ConfirmReservation(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toResponse(res))
}

func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	res, err := h.svc.CancelReservation(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toResponse(res))
}

func (h *Handler) getReservation(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	res, ok := h.svc.GetReservation(r.Context(), id)
	if !ok {
		writeError(w, http.StatusNotFound, domain.ErrReservationNotFound.Error())
		return
	}
	writeJSON(w, http.StatusOK, toResponse(res))
}

func (h *Handler) stock(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
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
