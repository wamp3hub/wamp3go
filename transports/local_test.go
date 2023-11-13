package wampTransports_test

import (
	"testing"

	wamp "github.com/wamp3hub/wamp3go"
	wampTransports "github.com/wamp3hub/wamp3go/transports"
)

func TestLocalTransportClose(t *testing.T) {
	left, _ := wampTransports.NewDuplexLocalTransport(10)

	e := left.Close()
	if e != nil {
		t.Errorf("Close failed: %v", e)
	}
}

func TestLocalTransportWrite(t *testing.T) {
	left, right := wampTransports.NewDuplexLocalTransport(10)
	event := wamp.MakeAcceptEvent("test", &wamp.AcceptFeatures{})

	e := left.Write(event)
	if e != nil {
		t.Errorf("Write failed: %v", e)
	}

	received, _ := right.Read()
	if received.ID() != event.ID() {
		t.Errorf("Write failed: expected %v, got %v", event.ID(), received.ID())
	}
}

func TestLocalTransportRead(t *testing.T) {
	left, right := wampTransports.NewDuplexLocalTransport(10)
	event := wamp.MakeAcceptEvent("test", &wamp.AcceptFeatures{})

	left.Write(event)

	received, e := right.Read()
	if e != nil {
		t.Errorf("Read failed: %v", e)
	}
	if received.ID() != event.ID() {
		t.Errorf("Read failed: expected %v, got %v", event.ID(), received.ID())
	}
}
