package rwlock

import "sync"

type Rwlock[T any] struct {
	Value T
	sync.RWMutex
}

// Get writer value and return unlocker function
//
// if call this function before end call function
func (rw *Rwlock[T]) Write() (T, func()) {
	rw.Lock()
	return rw.Value, rw.Unlock
}

// Get reader value and unlocker function
func (rw *Rwlock[T]) Read() (T, func()) {
	rw.RLock()
	return rw.Value, rw.RUnlock
}