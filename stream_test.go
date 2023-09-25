package wamp3go

import (
	"testing"
)

func TestHappyPathStream(t *testing.T) {
	stream := NewStream[string]()
	stream.Consume(
		func(v string) { t.Logf("alpha: %s", v) },
		func() { t.Logf("alpha: done") },
		1,
	)
	stream.Consume(
		func(v string) { t.Logf("beta: %s", v) },
		func() { t.Logf("beta: done") },
		1,
	)
	stream.Produce("Hello, world!")
	stream.Produce("Its working!")
	stream.Produce("Thats all")
	stream.Reset()
}
