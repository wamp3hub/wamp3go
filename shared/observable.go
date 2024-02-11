package wampShared

import "sync"

type nextFunction[T any] func(T)

type completeFunction func()

type Observer[T any] struct {
	next     nextFunction[T]
	complete completeFunction
}

// TODO unconsume

type Observable[T any] struct {
	observers []*Observer[T]
	mutex     sync.Mutex
}

func NewObservable[T any]() *Observable[T] {
	return new(Observable[T])
}

func (object *Observable[T]) Observe(next nextFunction[T], complete completeFunction) *Observer[T] {
	object.mutex.Lock()
	observer := Observer[T]{next, complete}
	object.observers = append(object.observers, &observer)
	object.mutex.Unlock()
	return &observer
}

func (object *Observable[T]) Next(v T) {
	for _, instance := range object.observers {
		instance.next(v)
	}
}

func (object *Observable[T]) Complete() {
	object.mutex.Lock()
	for _, instance := range object.observers {
		instance.complete()
	}
	clear(object.observers)
}

func (object *Observable[T]) Iterator(buffer int) <-chan T {
	q := make(chan T, buffer)

	object.Observe(
		func(v T) {
			q <- v
		},
		func() {
			close(q)
		},
	)

	return q
}
