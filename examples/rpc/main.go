package main

import (
	"errors"
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
		"example.greeting",
		&wamp.RegisterOptions{},
		func(callEvent wamp.CallEvent) wamp.ReplyEvent {
			name := ""
			e := callEvent.Payload(&name)
			if e == nil {
				return wamp.NewReplyEvent(callEvent, "Hello, "+name+"!")
			}
			return wamp.NewErrorEvent(callEvent, errors.New("InvalidName"))
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
