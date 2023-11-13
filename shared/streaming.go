package wampShared

import "sync"

type nextFunction[T any] func(T)
type completeFunction func()

type Consumer[T any] struct {
	next     nextFunction[T]
	complete completeFunction
}

// TODO unconsume

type Consumers[T any] struct {
	list  []Consumer[T]
	mutex sync.RWMutex
}

type Consumable[T any] func(nextFunction[T], completeFunction) *Consumer[T]
type Producible[T any] func(T)
type Closeable func()

func NewStream[T any]() (Consumable[T], Producible[T], Closeable) {
	consumers := Consumers[T]{}

	consume := func(next nextFunction[T], complete completeFunction) *Consumer[T] {
		instance := Consumer[T]{next, complete}
		consumers.mutex.Lock()
		consumers.list = append(consumers.list, instance)
		consumers.mutex.Unlock()
		return &instance
	}

	produce := func(v T) {
		// TODO rate limiting
		for _, instance := range consumers.list {
			go instance.next(v)
		}
	}

	close := func() {
		for _, instance := range consumers.list {
			go instance.complete()
		}
		consumers.mutex.Lock()
		consumers = Consumers[T]{}
		consumers.mutex.Lock()
	}

	return consume, produce, close
}
