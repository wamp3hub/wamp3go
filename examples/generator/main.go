package main

import (
	"fmt"

	wamp "github.com/wamp3hub/wamp3go"
	wampSerializer "github.com/wamp3hub/wamp3go/serializer"
	wampTransport "github.com/wamp3hub/wamp3go/transport"
)

type LoginPayload struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func reverse(callEvent wamp.CallEvent) wamp.ReplyEvent {
	var n int
	e := callEvent.Payload(&n)
	if e == nil {
		for i := n; i > 0; i-- {
			wamp.Yield(callEvent, i)
		}
		return wamp.NewReplyEvent(callEvent, 0)
	}
	return wamp.NewErrorEvent(callEvent, e)
}

func main() {
	session, e := wampTransport.WebsocketJoin(
		"0.0.0.0:8888",
		&wampSerializer.DefaultJSONSerializer,
		&LoginPayload{"test", "test"},
	)
	if e == nil {
		fmt.Printf("WAMP Join Success\n")
	} else {
		panic("WAMP Join Error")
	}

	registration, e := session.Register("example.reverse", &wamp.RegisterOptions{}, reverse)
	if e == nil {
		fmt.Printf("registration ID=%s\n", registration.ID)
	} else {
		panic("RegisterError")
	}

	callEvent := wamp.NewCallEvent(&wamp.CallFeatures{"example.reverse"}, 99)
	generator := session.Call(callEvent)

	var v int
	for {
		yieldEvent := wamp.Next(generator, wamp.DEFAULT_TIMEOUT)
		e := yieldEvent.Error()
		if e != nil {
			fmt.Printf("error(example.reversed): %s\n", e)
			break
		} else if yieldEvent.Done() {
			fmt.Print("generator done\n")
			break
		} else {
			yieldEvent.Payload(&v)
			fmt.Printf("call(example.reversed): %d\n", v)
		}
	}
}
