package shared

import (
	"sync"
	"testing"
)

func TestHappyPathStream(t *testing.T) {
	producer, consumer := NewStream[string]()
	wg := new(sync.WaitGroup)

	consumer.Consume(
		func(v string) {
			println("alpha: ", v)
			wg.Done()
		},
		func() {
			println("alpha: complete")
			wg.Done()
		},
	)
	consumer.Consume(
		func(v string) {
			println("beta: ", v)
			wg.Done()
		},
		func() {
			println("beta: complete")
			wg.Done()
		},
	)

	wg.Add(4)
	producer.Produce("Hello, world")
	producer.Produce("Nice")
	wg.Wait()

	wg.Add(2)
	producer.Close()
	wg.Wait()
}
