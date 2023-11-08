package wampShared

import (
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
