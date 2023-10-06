package shared

type nextFunction[T any] func(T)
type completeFunction func()

type Consumer[T any] struct {
	next     nextFunction[T]
	complete completeFunction
}

// TODO unconsume

type ConsumerList[T any] []Consumer[T]

type Consumable[T any] func(nextFunction[T], completeFunction) *Consumer[T]
type Producible[T any] func(T)
type Closeable func()

func NewStream[T any]() (Consumable[T], Producible[T], Closeable) {
	consumerList := ConsumerList[T]{}

	consume := func(next nextFunction[T], complete completeFunction) *Consumer[T] {
		consumer := Consumer[T]{next, complete}
		consumerList = append(consumerList, consumer)
		return &consumer
	}

	produce := func(v T) {
		// TODO rate limiting
		for _, consumer := range consumerList {
			go consumer.next(v)
		}
	}

	close := func() {
		for _, consumer := range consumerList {
			go consumer.complete()
		}
		consumerList = ConsumerList[T]{}
	}

	return consume, produce, close
}
