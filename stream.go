package wamp3go

import "sync"

type consumerList[T any] []chan T

type Stream[T any] struct {
	consumers consumerList[T]
}

func NewStream[T any]() *Stream[T] {
	return &Stream[T]{consumerList[T]{}}
}

func (stream *Stream[T]) ConsumerCount() int {
	return len(stream.consumers)
}

func consume[T any](
	wg *sync.WaitGroup,
	consumer chan T,
	next func(T),
	complete func(),
) {
	wg.Done()
	for v := range consumer {
		go next(v)
	}
	complete()
}

func (stream *Stream[T]) Consume(
	next func(T),
	complete func(),
	size int,
) {
	consumer := make(chan T, size)
	stream.consumers = append(stream.consumers, consumer)
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go consume(wg, consumer, next, complete)
	wg.Wait()
}

func (stream *Stream[T]) Produce(data T) {
	for _, consumer := range stream.consumers {
		consumer <- data
	}
}

func (stream *Stream[T]) Reset() {
	for _, consumer := range stream.consumers {
		close(consumer)
	}
	stream.consumers = consumerList[T]{}
}
