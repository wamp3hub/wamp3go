package shared

import (
	"errors"
	"time"
)

type promise[T any] <-chan T

func NewPromise[T any](timeout time.Duration) (promise[T], func(T), func()) {
	instance := make(chan T, 1)

	timer := new(time.Timer)

	cancel := func() {
		timer.Stop()
		close(instance)
	}

	timer = time.AfterFunc(timeout, cancel)

	complete := func(value T) {
		instance <- value
		cancel()
	}

	return instance, complete, cancel
}

type PendingMap[T any] map[string]func(T)

func NewPendingMap[T any]() PendingMap[T] {
	return make(PendingMap[T])
}

func (pendingMap PendingMap[T]) New(
	key string,
	timeout time.Duration,
) promise[T] {
	__promise, complete, _ := NewPromise[T](timeout)
	pendingMap[key] = complete
	return __promise
}

func (pendingMap PendingMap[T]) Complete(key string, value T) error {
	complete, found := pendingMap[key]
	if found {
		complete(value)
		return nil
	}
	return errors.New("PendingNotFound")
}
