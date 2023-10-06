package shared

import (
	"sync"
	"testing"
)

func TestHappyPathStream(t *testing.T) {
	consume, produce, close := NewStream[string]()
	wg := new(sync.WaitGroup)

	consume(
		func(v string) {
			println("alpha: ", v)
			wg.Done()
		},
		func() {
			println("alpha: complete")
			wg.Done()
		},
	)
	consume(
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
	produce("Hello, world")
	produce("Nice")
	wg.Wait()

	wg.Add(2)
	close()
	wg.Wait()
}
