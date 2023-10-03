package main

import (
	"fmt"

	wamp "github.com/wamp3hub/wamp3go"
	wampSerializer "github.com/wamp3hub/wamp3go/serializer"
	wampTransport "github.com/wamp3hub/wamp3go/transport"
)

func main() {
	session, e := wampTransport.WebsocketJoin(
		"0.0.0.0:9999",
		&wampSerializer.DefaultJSONSerializer,
	)
	if e == nil {
		fmt.Printf("WAMP Join Success\n")
	} else {
		panic("WAMP Join Error")
	}

	reversed := func(callEvent wamp.CallEvent) wamp.ReplyEvent {
		n := 0
		e := callEvent.Payload(&n)
		fmt.Printf("reverse %d\n", n)
		if e == nil {
			for i := n; i > 0; i-- {
				session.Yield(callEvent, i)
			}
			replyEvent := wamp.NewReplyEvent(callEvent.ID(), 0)
			return replyEvent
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
		replyEvent := session.Next(generator, wamp.DEFAULT_TIMEOUT)
		replyFeatures := replyEvent.Features()
		if replyFeatures.OK {
			var v int
			replyEvent.Payload(&v)
			fmt.Printf("call(example.reversed) %d\n", v)
		} else {
			e = wamp.ExtractError(replyEvent)
			fmt.Printf("call(example.reversed) %s\n", e)
		}
	}

	fmt.Print("generator done\n")
}
