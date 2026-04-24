package infrastructure

import (
	"sync"
	"time"
)

// Clock abstracts time to keep domain logic deterministic in tests.
type Clock interface {
	Now() time.Time
}

// SystemClock returns the real wall clock time.
type SystemClock struct{}

// Now returns time.Now().
func (SystemClock) Now() time.Time { return time.Now() }

// FakeClock is a test clock whose time is advanced manually.
type FakeClock struct {
	mu      sync.Mutex
	current time.Time
}

// NewFakeClock creates a FakeClock pinned at t.
func NewFakeClock(t time.Time) *FakeClock { return &FakeClock{current: t} }

// Now returns the clock's current time.
func (c *FakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current
}

// Advance moves the clock forward by d.
func (c *FakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.current = c.current.Add(d)
}
