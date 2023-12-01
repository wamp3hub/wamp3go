package main

import (
	"fmt"

	wamp "github.com/wamp3hub/wamp3go"
	wampSerializers "github.com/wamp3hub/wamp3go/serializers"
	wampTransports "github.com/wamp3hub/wamp3go/transports"
)

func main() {
	type LoginPayload struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	session, e := wampTransports.WebsocketJoin(
		"0.0.0.0:8888",
		false,
		wampSerializers.DefaultSerializer,
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
					source, _ = wamp.Yield(source, i)
				}
				return wamp.NewReplyEvent(source, 0)
			}
			return wamp.GeneratorExit(source)
		},
	)
	if e == nil {
		fmt.Printf("register success ID=%s\n", registration.ID)
	} else {
		panic("register error")
	}

	generator, e := wamp.NewRemoteGenerator[int](
		session,
		&wamp.CallFeatures{URI: "example.reverse"},
		99,
	)
	if e != nil {
		fmt.Printf("generator create error %s\n", e)
		panic("generator create error")
	}

	for generator.Active() {
		fmt.Print("call(example.reversed): ")
		_, v, e := generator.Next(wamp.DEFAULT_TIMEOUT)
		if e == nil {
			fmt.Printf("%d\n", v)
		} else {
			fmt.Printf("error %s\n", e)
		}
	}
	fmt.Print("generator done\n")
}
