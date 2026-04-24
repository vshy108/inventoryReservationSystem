package domain_test

import (
	"testing"

	"everest/inventoryReservation/internal/domain"
)

func TestReservation_IsActive(t *testing.T) {
	cases := []struct {
		state domain.ReservationState
		want  bool
	}{
		{domain.StateActive, true},
		{domain.StateConfirmed, false},
		{domain.StateCancelled, false},
		{domain.StateExpired, false},
	}
	for _, c := range cases {
		if got := (domain.Reservation{State: c.state}).IsActive(); got != c.want {
			t.Errorf("state=%s: want %v, got %v", c.state, c.want, got)
		}
	}
}
