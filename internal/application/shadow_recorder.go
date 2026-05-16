package application

import (
	"context"

	"everest/inventoryReservation/internal/domain"
)

type ShadowAction string

const (
	ShadowReserved  ShadowAction = "Reserved"
	ShadowConfirmed ShadowAction = "Confirmed"
	ShadowCancelled ShadowAction = "Cancelled"
	ShadowExpired   ShadowAction = "Expired"
)

type ShadowRecord struct {
	Action      ShadowAction
	Product     domain.ProductInventory
	Reservation domain.Reservation
	Event       domain.InventoryEvent
}

type ShadowRecorder interface {
	RecordShadow(context.Context, ShadowRecord)
}

type NoopShadowRecorder struct{}

func (NoopShadowRecorder) RecordShadow(context.Context, ShadowRecord) {}
