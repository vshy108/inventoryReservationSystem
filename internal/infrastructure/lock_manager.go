package infrastructure

import "sync"

// LockManager provides a named mutex keyed by an arbitrary string
// (for example a product ID). It is used to serialize the
// read-check-write sequence of reservation operations per product.
type LockManager struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

// NewLockManager creates an empty LockManager.
func NewLockManager() *LockManager {
	return &LockManager{locks: make(map[string]*sync.Mutex)}
}

func (lm *LockManager) lockFor(key string) *sync.Mutex {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	l, ok := lm.locks[key]
	if !ok {
		l = &sync.Mutex{}
		lm.locks[key] = l
	}
	return l
}

// With acquires the lock for key, runs fn, and releases the lock.
func (lm *LockManager) With(key string, fn func()) {
	l := lm.lockFor(key)
	l.Lock()
	defer l.Unlock()
	fn()
}
