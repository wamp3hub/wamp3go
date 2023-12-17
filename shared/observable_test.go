package wampShared_test

import (
	"sync"
	"testing"

	wampShared "github.com/wamp3hub/wamp3go/shared"
)

func TestObservableHappyPath(t *testing.T) {
	wg := new(sync.WaitGroup)

	events := wampShared.NewObservable[string]()

	events.Observe(
		func(v string) {
			t.Logf("alpha: %s", v)
			wg.Done()
		},
		func() {
			t.Log("alpha: complete")
			wg.Done()
		},
	)

	events.Observe(
		func(v string) {
			t.Logf("beta: %s", v)
			wg.Done()
		},
		func() {
			t.Log("beta: complete")
			wg.Done()
		},
	)

	testData := []string{
		"Hi!",
		"How are you?",
		"Nice to meet you",
		"Test",
		"Goodbye",
	}

	wg.Add(len(testData) * 2)
	for _, v := range testData {
		events.Next(v)
	}
	wg.Wait()

	wg.Add(2)
	events.Complete()
	wg.Wait()
}
