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
	n := 0
	e := callEvent.Payload(&n)
	if e == nil {
		for i := n; i > 0; i-- {
			e = wamp.Yield(callEvent, i)
			if e != nil {
				fmt.Printf("YieldError %s", e)
				break
			}
		}
		return wamp.NewReplyEvent(callEvent, 0)
	}
	return wamp.NewErrorEvent(callEvent, e)
}

func createSession() *wamp.Session {
	session, e := wampTransport.WebsocketJoin(
		"0.0.0.0:8888",
		&wampSerializer.DefaultSerializer,
		&LoginPayload{"test", "test"},
	)
	if e == nil {
		fmt.Printf("WAMP Join Success\n")
		return session
	} else {
		panic("WAMP Join Error")
	}
}

func main() {
	asession := createSession()
	registration, e := wamp.Register(asession, "example.reverse", &wamp.RegisterOptions{}, reverse)
	if e == nil {
		fmt.Printf("registration ID=%s\n", registration.ID)
	} else {
		panic("RegisterError")
	}

	bsession := createSession()
	generator, e := wamp.NewGenerator[int](bsession, &wamp.CallFeatures{"example.reverse"}, 99)
	if e == nil {
		fmt.Printf("reverse generator created\n")
	} else {
		panic("failed to create generator")
	}
	for !generator.Done() {
		fmt.Print("call(example.reversed): ")
		result := generator.Next(wamp.DEFAULT_TIMEOUT)
		_, v, e := result.Await()
		if e == nil {
			fmt.Printf("%d\n", v)
		} else {
			fmt.Printf("error %s\n", e)
		}
	}
	fmt.Print("generator done\n")
}
