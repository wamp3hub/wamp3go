package main

import (
	"fmt"

	wamp "github.com/wamp3hub/wamp3go"
	wampTransports "github.com/wamp3hub/wamp3go/transports"
)

func main() {
	type LoginPayload struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	session, e := wampTransports.WebsocketJoin(
		&wampTransports.WebsocketJoinOptions{
			Address:     "0.0.0.0:8800",
			Credentials: &LoginPayload{"test", "test"},
		},
	)
	if e == nil {
		fmt.Printf("WAMP Join Success\n")
	} else {
		panic("WAMP Join Error")
	}

	registration, e := wamp.Register(
		session,
		"net.example.reverse",
		&wamp.RegisterOptions{},
		func(n int, callEvent wamp.CallEvent) (int, error) {
			source := wamp.Event(callEvent)
			for i := n; i > 0; i-- {
				source = wamp.Yield(source, i)
			}
			return -1, wamp.GeneratorExit(source)
		},
	)
	if e == nil {
		fmt.Printf("register success ID=%s\n", registration.ID)
	} else {
		panic("register error")
	}

	generator, e := wamp.NewRemoteGenerator[int](
		session,
		&wamp.CallFeatures{URI: "net.example.reverse"},
		100,
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
