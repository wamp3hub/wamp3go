package wampShared

import (
	"testing"
	"time"
)

func TestPromiseComplete(t *testing.T) {
	promise, complete, _ := NewPromise[string](time.Second)
	complete("Hello, world")
	v, ok := <-promise
	t.Logf("v=%s ok=%t", v, ok)
}

func TestPromiseCancel(t *testing.T) {
	promise, _, cancel := NewPromise[string](time.Second)
	cancel()
	v, ok := <-promise
	t.Logf("v=%s ok=%t", v, ok)
}

func TestPromiseTimeout(t *testing.T) {
	promise, _, _ := NewPromise[string](time.Second)
	v, ok := <-promise
	t.Logf("v=%s ok=%t", v, ok)
}

func TestHappyPathPendingMap(t *testing.T) {
	instance := make(PendingMap[bool])

	for _, key := range []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta"} {
		promise, _ := instance.New(key, 60*time.Second)
		go instance.Complete(key, true)
		v, ok := <-promise
		if ok && v == true {
			t.Log("OK")
		} else {
			t.Log("AssertionError")
		}
	}
}
