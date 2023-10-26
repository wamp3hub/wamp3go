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
		"example.greeting",
		&wamp.RegisterOptions{},
		func(callEvent wamp.CallEvent) wamp.ReplyEvent {
			name := ""
			callEvent.Payload(&name)
			return wamp.NewReplyEvent(callEvent, "Hello, "+name+"!")
		},
	)
	if e == nil {
		fmt.Printf("registration ID=%s\n", registration.ID)
	} else {
		panic("RegisterError")
	}

	pendingResponse := wamp.Call[string](session, &wamp.CallFeatures{URI: "example.greeting"}, "WAMP")
	_, v, e := pendingResponse.Await()
	if e == nil {
		fmt.Printf("call(example.greeting) %s\n", v)
	} else {
		fmt.Printf("call(example.greeting) %s\n", e)
	}
}
