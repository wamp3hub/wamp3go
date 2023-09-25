package wamp3go

import (
	"sync"
	"testing"
	"time"
)

func assertPending[T comparable](
	t *testing.T,
	wg *sync.WaitGroup,
	instance *PendingMap[T],
	key string,
	timeout time.Duration,
	expectedValue T,
) {
	wg.Done()
	v, e := instance.Catch(key, timeout)
	if e != nil {
		t.Logf("ERROR: %s", e)
	} else if v != expectedValue {
		t.Logf("AssertionError")
	} else {
		t.Logf("OK")
	}
	wg.Done()
}

func TestHappyPathPendingMap(t *testing.T) {
	instance := make(PendingMap[bool])
	wg := sync.WaitGroup{}

	for _, key := range []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta"} {
		wg.Add(1)
		go assertPending(t, &wg, &instance, key, DEFAULT_TIMEOUT, true)
		wg.Wait()
		wg.Add(1)
		instance.Throw(key, true)
		wg.Wait()
	}
}

func TestTimeoutPendingMap(t *testing.T) {
	instance := make(PendingMap[bool])
	wg := sync.WaitGroup{}

	wg.Add(2)
	go assertPending(t, &wg, &instance, "test", time.Second, true)
	wg.Wait()
}
