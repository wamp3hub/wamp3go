package main

import (
	"fmt"

	wamp "github.com/wamp3hub/wamp3go"
	wampSerializer "github.com/wamp3hub/wamp3go/serializer"
	wampTransport "github.com/wamp3hub/wamp3go/transport"
)

func main() {
	type LoginPayload struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	session, e := wampTransport.WebsocketJoin(
		"0.0.0.0:8888",
		&wampSerializer.DefaultSerializer,
		&LoginPayload{"test", "test"},
	)
	if e == nil {
		fmt.Printf("WAMP Join Success\n")
	} else {
		panic("WAMP Join Error")
	}

	registration, e := wamp.Register(
		session,
		"example.reverse",
		&wamp.RegisterOptions{},
		func(callEvent wamp.CallEvent) wamp.ReplyEvent {
			source := wamp.Event(callEvent)
			n := 0
			e := callEvent.Payload(&n)
			if e == nil {
				for i := n; i > 0; i-- {
					source, e = wamp.Yield(source, i)
					if e != nil {
						fmt.Printf("YieldError %s", e)
						break
					}
				}
				return wamp.NewReplyEvent(source, 0)
			}
			return wamp.NewErrorEvent(source, e)
		},
	)
	if e == nil {
		fmt.Printf("registration ID=%s\n", registration.ID)
	} else {
		panic("RegisterError")
	}

	generator := wamp.NewRemoteGenerator[int](session, &wamp.CallFeatures{URI: "example.reverse"}, 99)
	for !generator.Done() {
		fmt.Print("call(example.reversed): ")
		v, e := generator.Next(wamp.DEFAULT_TIMEOUT)
		if e == nil {
			fmt.Printf("%d\n", v)
		} else {
			fmt.Printf("error %s\n", e)
		}
	}
	fmt.Print("generator done\n")
}
