package wampShared_test

import (
	"sync"
	"testing"
	"time"

	wampShared "github.com/wamp3hub/wamp3go/shared"
)

func TestPendingMap(t *testing.T) {
	completable := func(t *testing.T, wg *sync.WaitGroup, instance *wampShared.PendingMap[string]) {
		key := wampShared.NewID()
        expectedResult := "Hello, WAMP!"

		promise, _ := instance.New(key, time.Minute)

        completeLater := func() {
            time.Sleep(time.Second)
            instance.Complete(key, expectedResult)
        }
        go completeLater()

        result, done := <-promise
		if !done {
			t.Errorf("Promise was not completed")
		}
        if result != expectedResult {
            t.Errorf("Expected %v, but got %v", expectedResult, result)
        }

		wg.Done()
    }

	cancellable := func(t *testing.T, wg *sync.WaitGroup, instance *wampShared.PendingMap[string]) {
		key := wampShared.NewID()

		promise, cancel := instance.New(key, time.Minute)

		cancelLater := func() {
			time.Sleep(time.Second)
			cancel()
		}
		go cancelLater()

		result, done := <-promise
		if done {
			t.Errorf("invalid behaviour result=%s", result)
		}

		wg.Done()
	}

	timedOut := func(t *testing.T, wg *sync.WaitGroup, instance *wampShared.PendingMap[string]) {
		key := wampShared.NewID()

		promise, _ := instance.New(key, time.Second)

		result, done := <-promise
		if done {
			t.Errorf("invalid behaviour result=%s", result)
		}

		wg.Done()
	}

	notFound := func(t *testing.T, wg *sync.WaitGroup, instance *wampShared.PendingMap[string]) {
		key := wampShared.NewID()

		e := instance.Complete(key, "not-found")
		if e == nil {
			t.Errorf("invalid behaviour")
		}

		wg.Done()
	}

	t.Run(
		"Case: Completable",
		func(t *testing.T) {
			wg := new(sync.WaitGroup)
			wg.Add(1)
			completable(t, wg, wampShared.NewPendingMap[string]())
			wg.Wait()
		},
	)

	t.Run(
		"Case: Cancellable",
		func(t *testing.T) {
			wg := new(sync.WaitGroup)
			wg.Add(1)
			cancellable(t, wg, wampShared.NewPendingMap[string]())
			wg.Wait()
		},
	)

	t.Run(
		"Case: TimedOut",
		func(t *testing.T) {
			wg := new(sync.WaitGroup)
			wg.Add(1)
			timedOut(t, wg, wampShared.NewPendingMap[string]())
			wg.Wait()
		},
	)

	t.Run(
		"Case: NotFound",
		func(t *testing.T) {
			wg := new(sync.WaitGroup)
			wg.Add(1)
			notFound(t, wg, wampShared.NewPendingMap[string]())
			wg.Wait()
		},
	)

	t.Run(
		"Case: Complex",
		func(t *testing.T) {
			wg := new(sync.WaitGroup)
			n := 100
			wg.Add(n * 4)
			instance := wampShared.NewPendingMap[string]()
			for i := 0; i < n; i++ {
				go completable(t, wg, instance)
				go cancellable(t, wg, instance)
				go timedOut(t, wg, instance)
				go notFound(t, wg, instance)
			}
			wg.Wait()
		},
	)
}
