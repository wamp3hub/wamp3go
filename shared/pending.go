package shared

import (
	"errors"
	"time"
)

type promise[T any] chan T

type PendingMap[T any] map[string]promise[T]

func NewPendingMap[T any]() PendingMap[T] {
	return make(PendingMap[T])
}

func (pendingMap PendingMap[T]) Catch(
	key string,
	timeout time.Duration,
) (value T, e error) {
	__promise, exist := pendingMap[key]
	if !exist {
		__promise = make(promise[T])
		pendingMap[key] = __promise
	}
	select {
	case value = <-__promise:
	case <-time.After(timeout):
		e = errors.New("TimedOut")
	}
	delete(pendingMap, key)
	close(__promise)
	return value, e
}

func (pendingMap PendingMap[T]) Throw(key string, value T) error {
	promise, exist := pendingMap[key]
	if exist {
		promise <- value
		return nil
	}
	return errors.New("PendingNotFound")
}
