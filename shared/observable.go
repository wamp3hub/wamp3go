package wampShared

import "sync"

type nextFunction[T any] func(T)

type completeFunction func()

type Observer[T any] struct {
	next     nextFunction[T]
	complete completeFunction
}

// TODO unconsume

type ObservableObject[T any] struct {
	done      bool
	observers []*Observer[T]
	mutex     sync.RWMutex
}

func (object *ObservableObject[T]) Observe(next nextFunction[T], complete completeFunction) *Observer[T] {
	object.mutex.Lock()
	observer := Observer[T]{next, complete}
	object.observers = append(object.observers, &observer)
	object.mutex.Unlock()
	return &observer
}

func (object *ObservableObject[T]) Next(v T) {
	// TODO rate limiting
	for _, instance := range object.observers {
		go instance.next(v)
	}
}

func (object *ObservableObject[T]) Complete() {
	object.mutex.Lock()
	object.done = true
	for _, instance := range object.observers {
		go instance.complete()
	}
	clear(object.observers)
}

func NewObservable[T any]() *ObservableObject[T] {
	return new(ObservableObject[T])
}
