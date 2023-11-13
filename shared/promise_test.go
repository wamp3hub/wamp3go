package wampShared_test

import (
	"testing"
	"time"

	wampShared "github.com/wamp3hub/wamp3go/shared"
)

func TestPromise(t *testing.T) {
	t.Run("Case: Completable", func(t *testing.T) {
		promise, completePromise, _ := wampShared.NewPromise[string](time.Minute)

		expectedResult := "Hello, WAMP!"

		completeLater := func() {
			time.Sleep(time.Second)
			completePromise(expectedResult)
		}
		go completeLater()

		result, done := <-promise
		if !done {
			t.Errorf("Promise was not completed")
		}
		if result != expectedResult {
			t.Errorf("Expected %v, but got %v", expectedResult, result)
		}
	})

	t.Run("Case: Cancellable", func(t *testing.T) {
		promise, _, cancelPromise := wampShared.NewPromise[string](time.Minute)

		cancelLater := func() {
			time.Sleep(time.Second)
			cancelPromise()
		}
		go cancelLater()

		result, done := <-promise
		if done {
			t.Errorf("invalid behaviour result=%s", result)
		}
	})

	t.Run("Case: TimedOut", func(t *testing.T) {
		promise, _, cancelPromise := wampShared.NewPromise[string](time.Second)

		result, done := <-promise
		if done {
			t.Errorf("invalid behaviour result=%s", result)
		}

        cancelPromise()
	})
}
