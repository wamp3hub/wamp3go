package wamp3go

import (
	"errors"
	"time"
)

const DEFAULT_TIMEOUT = 60 * time.Second

type PendingMap[T any] map[string]chan T

func (pendingMap PendingMap[T]) Catch(
	key string,
	timeout time.Duration,
) (value T, e error) {
	promise, exist := pendingMap[key]
	if !exist {
		promise = make(chan T)
		pendingMap[key] = promise
	}
	select {
	case value = <-promise:
	case <-time.After(timeout):
		e = errors.New("TimedOut")
	}
	delete(pendingMap, key)
	close(promise)
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
