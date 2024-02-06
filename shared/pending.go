package wampShared

import (
	"errors"
	"time"

	cmap "github.com/orcaman/concurrent-map/v2"
)

var ErrorPendingNotFound = errors.New("PendingNotFound")

type PendingMap[T any] struct {
	safeMap cmap.ConcurrentMap[string, CompletePromise[T]]
}

func NewPendingMap[T any]() *PendingMap[T] {
	return &PendingMap[T]{
		cmap.New[CompletePromise[T]](),
	}
}

func (pendingMap PendingMap[T]) New(
	key string,
	timeout time.Duration,
) (Promise[T], CancelPromise) {
	_, ok := pendingMap.safeMap.Get(key)
	if ok {
		// Instead of using panic when a pending already exists, consider returning an error.
		// This would allow the caller to decide how to handle this situation.
		panic("pending already exists")
	}

	promise, completePromise, cancelPromise := NewPromise[T](timeout)

	pendingMap.safeMap.Set(key, completePromise)

	cancelPending := func() {
		pendingMap.safeMap.Remove(key)
		cancelPromise()
	}

	return promise, cancelPending
}

func (pendingMap PendingMap[T]) Complete(key string, value T) error {
	completePending, exists := pendingMap.safeMap.Get(key)
	if exists {
		pendingMap.safeMap.Remove(key)
		completePending(value)
		return nil
	}
	return ErrorPendingNotFound
}
