package shared

import (
	"errors"
	"sync"
	"time"
)

type Promise[T any] <-chan T

type Completable[T any] func(T)

type Cancellable func()

func NewPromise[T any](timeout time.Duration) (Promise[T], Completable[T], Cancellable) {
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
