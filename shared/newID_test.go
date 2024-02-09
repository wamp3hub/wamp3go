package wampShared_test

import (
	"testing"

	wampShared "github.com/wamp3hub/wamp3go/shared"
)

func TestNewID(t *testing.T) {
	t.Run("Case: Check uniqueness", func(t *testing.T) {
		set := make(map[string]struct{})
		for i := 0; i < 1000000; i++ {
			v := wampShared.NewID()
			_, ok := set[v]
			if ok {
				t.Errorf("NewID Error: %v is not unique", v)
			}
			set[v] = struct{}{}
		}
	})
}
