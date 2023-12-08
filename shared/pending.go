package wampShared

import (
	"errors"
	"sync"
	"time"
)

var ErrorPendingNotFound = errors.New("PendingNotFound")

type PendingMap[T any] struct {
	safeMap map[string]CompletePromise[T]
	mutex   *sync.RWMutex
}

func NewPendingMap[T any]() *PendingMap[T] {
	return &PendingMap[T]{
		make(map[string]CompletePromise[T]),
		new(sync.RWMutex),
	}
}

func (pendingMap PendingMap[T]) safeGet(key string) (CompletePromise[T], bool) {
	pendingMap.mutex.RLock()
	defer pendingMap.mutex.RUnlock()
	value, exists := pendingMap.safeMap[key]
	return value, exists
}

func (pendingMap PendingMap[T]) safeSet(key string, value CompletePromise[T]) {
	pendingMap.mutex.Lock()
	pendingMap.safeMap[key] = value
	pendingMap.mutex.Unlock()
}

func (pendingMap PendingMap[T]) safeDelete(key string) {
	pendingMap.mutex.Lock()
	delete(pendingMap.safeMap, key)
	pendingMap.mutex.Unlock()
}

func (pendingMap PendingMap[T]) New(
	key string,
	timeout time.Duration,
) (Promise[T], CancelPromise) {
	_, exists := pendingMap.safeGet(key)
	if exists {
		// Instead of using panic when a pending already exists, consider returning an error.
		// This would allow the caller to decide how to handle this situation.
		panic("pending already exists")
	}

	pending, completePromise, cancelPromise := NewPromise[T](timeout)

	pendingMap.safeSet(key, completePromise)

	cancelPending := func() {
		pendingMap.safeDelete(key)
		cancelPromise()
	}

	return pending, cancelPending
}

func (pendingMap PendingMap[T]) Complete(key string, value T) error {
	completePending, exists := pendingMap.safeGet(key)
	if exists {
		pendingMap.safeDelete(key)
		completePending(value)
		return nil
	}
	return ErrorPendingNotFound
}
