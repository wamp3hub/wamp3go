package wampShared

import (
	"sync"
	"time"
)

type Promise[T any] <-chan T

type CompletePromise[T any] func(T)

type CancelPromise func()

func NewTimelessPromise[T any]() (Promise[T], CompletePromise[T], CancelPromise) {
	channel := make(chan T, 1)

	once := new(sync.Once)

	cancel := func() {
		closeChannel := func() {
			close(channel)
		}
		once.Do(closeChannel)
	}

	complete := func(value T) {
		channel <- value
		cancel()
	}

	return channel, complete, cancel
}

func NewPromise[T any](timeout time.Duration) (Promise[T], CompletePromise[T], CancelPromise) {
	promise, complete, cancel := NewTimelessPromise[T]()

	if timeout > 0 {
		await := func() {
			<-time.After(timeout)
			cancel()
		}

		go await()
	}

	return promise, complete, cancel
}
