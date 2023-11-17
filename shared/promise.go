package wampShared

import (
	"sync"
	"time"
)

type Promise[T any] <-chan T

type Completable[T any] func(T)

type Cancellable func()

func NewPromise[T any](timeout time.Duration) (Promise[T], Completable[T], Cancellable) {
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

	await := func() {
		<-time.After(timeout)
		cancel()
	}

	go await()

	return channel, complete, cancel
}

// func NewPromise[T any]() (Promise[T], Completable[T], Cancellable) {
// 	channel := make(chan T, 1)

// 	once := new(sync.Once)

// 	cancel := func() {
// 		closeChannel := func() {
// 			close(channel)
// 		}
// 		once.Do(closeChannel)
// 	}

// 	complete := func(value T) {
// 		channel <- value
// 		cancel()
// 	}

// 	return channel, complete, cancel
// }

// func NewPromise[T any](timeout time.Duration) (Promise[T], Completable[T], Cancellable) {
// 	promise, complete, cancel := NewPromise[T]()

// 	await := func() {
// 		<-time.After(timeout)
// 		cancel()
// 	}

// 	go await()

// 	return promise, complete, cancel
// }
