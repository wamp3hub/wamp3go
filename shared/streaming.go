package shared

type nextFunction[T any] func(T)
type completeFunction func()

type Listener[T any] struct {
	next     nextFunction[T]
	complete completeFunction
}

type ListenerList[T any] []Listener[T]

type Consumer[T any] struct {
	listeners ListenerList[T]
}

func (c *Consumer[T]) Consume(
	next nextFunction[T],
	complete completeFunction,
) {
	c.listeners = append(c.listeners, Listener[T]{next, complete})
}

func (c *Consumer[T]) Close() {
	for _, subscription := range c.listeners {
		go subscription.complete()
	}
	c.listeners = ListenerList[T]{}
}

type Producer[T any] struct {
	consumer *Consumer[T]
}

func (p *Producer[T]) Produce(data T) {
	for _, subscription := range p.consumer.listeners {
		go subscription.next(data)
	}
}

func (p *Producer[T]) Close() {
	p.consumer.Close()
}

func NewStream[T any]() (*Producer[T], *Consumer[T]) {
	listeners := ListenerList[T]{}
	c := Consumer[T]{listeners}
	p := Producer[T]{&c}
	return &p, &c
}
