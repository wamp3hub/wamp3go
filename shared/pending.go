package shared

import (
	"errors"
	"sync"
	"time"
)

type Promise[T any] <-chan T

func NewPromise[T any](timeout time.Duration) (Promise[T], func(T), func()) {
	instance := make(chan T, 1)

	once := new(sync.Once)
	cancel := func() { close(instance) }

	complete := func(value T) {
		instance <- value
		once.Do(cancel)
	}

	await := func() {
		<-time.After(timeout)
		once.Do(cancel)
	}

	go await()

	return instance, complete, cancel
}

type PendingMap[T any] map[string]func(T)

func NewPendingMap[T any]() PendingMap[T] {
	return make(PendingMap[T])
}

func (pendingMap PendingMap[T]) New(
	key string,
	timeout time.Duration,
) Promise[T] {
	__promise, complete, _ := NewPromise[T](timeout)
	pendingMap[key] = complete
	return __promise
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
