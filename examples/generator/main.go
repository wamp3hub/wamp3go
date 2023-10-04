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

	reversed := func(callEvent wamp.CallEvent) wamp.ReplyEvent {
		var n int
		e := callEvent.Payload(&n)
		if e == nil {
			for i := n; i > -1; i-- {
				session.Yield(callEvent, i)
			}
			return wamp.NewReplyEvent(callEvent.ID(), 0)
		}
		return wamp.NewErrorEvent(callEvent.ID(), e)
	}

	registration, e := session.Register("example.reversed", &wamp.RegisterOptions{}, reversed)
	if e == nil {
		fmt.Printf("registration ID=%s\n", registration.ID)
	} else {
		panic("RegisterError")
	}

	n := 99

	callEvent := wamp.NewCallEvent(&wamp.CallFeatures{"example.reversed"}, n)
	generator := session.Call(callEvent)

	for i := 0; i < n; i++ {
		yieldEvent := session.Next(generator, wamp.DEFAULT_TIMEOUT)
		replyFeatures := yieldEvent.Features()
		if replyFeatures.OK {
			var v int
			yieldEvent.Payload(&v)
			fmt.Printf("call(example.reversed) %d\n", v)
		} else {
			e = wamp.ExtractError(yieldEvent)
			fmt.Printf("call(example.reversed) %s\n", e)
		}
	}

	fmt.Print("generator done\n")
}
