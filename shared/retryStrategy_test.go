package wampShared_test

import (
	"testing"
	"time"

	wampShared "github.com/wamp3hub/wamp3go/shared"
)

func TestConstantRS(t *testing.T) {
	rs := wampShared.NewConstantRS(time.Second, 3)

	for i := 1; i < 4; i++ {
		v := rs.Next()
		if rs.AttemptNumber() != i || v != time.Second {
			t.Errorf("Next failed: expected attempt number %v and value 1s, got %v and %v", i, rs.AttemptNumber(), v)
		}
	}

	done := rs.Done()
	if !done {
		t.Errorf("Done failed: expected true, got %v", done)
	}

	rs.Reset()
	an := rs.AttemptNumber()
	if an != 0 {
		t.Errorf("Reset failed: expected attempt number 0, got %v", an)
	}
}

func TestBackoffRS(t *testing.T) {
	t.Run("Case: Happy Path", func(t *testing.T) {
		rs := wampShared.NewBackoffRS(time.Second, 2, time.Hour, 3)
		expectedValue := time.Second
		for i := 1; i < 4; i++ {
			v := rs.Next()
			if rs.AttemptNumber() != i || expectedValue > v {
				t.Errorf("Next failed: expected attempt number %v and value greater than %v, got %v and %v", i, expectedValue, rs.AttemptNumber(), v)
			}
			expectedValue = v
		}

		done := rs.Done()
		if !done {
			t.Errorf("Done failed: expected true, got %v", done)
		}

		rs.Reset()
		an := rs.AttemptNumber()
		if an != 0 {
			t.Errorf("Reset failed: expected attempt number 0, got %v", an)
		}
	})

	t.Run("Case: Upper Bound", func(t *testing.T) {
		expectedValue := time.Second
		rs := wampShared.NewBackoffRS(expectedValue, 2, expectedValue, 3)
		for i := 1; i < 4; i++ {
			v := rs.Next()
			if rs.AttemptNumber() != i || v > expectedValue {
				t.Errorf("Next failed: expected attempt number %v and value %v, got %v and %v", i, expectedValue, rs.AttemptNumber(), v)
			}
		}
	})
}
