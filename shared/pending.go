package shared

import (
	"errors"
	"time"
)

type PendingMap[T any] map[string]func(T)

func NewPendingMap[T any]() PendingMap[T] {
	return make(PendingMap[T])
}

func (pendingMap PendingMap[T]) New(
	key string,
	timeout time.Duration,
) (Promise[T], Cancellable) {
	promise, complete, cancelPromise := NewPromise[T](timeout)
	pendingMap[key] = complete
	cancel := func() {
		delete(pendingMap, key)
		cancelPromise()
	}
	return promise, cancel
}

func (pendingMap PendingMap[T]) Complete(key string, value T) error {
	complete, found := pendingMap[key]
	if found {
		complete(value)
		delete(pendingMap, key)
		return nil
	}
	return errors.New("PendingNotFound")
}
