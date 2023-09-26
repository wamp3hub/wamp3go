package main

import (
	"fmt"
	wamp "wamp3go"
	wamp_serializer "wamp3go/serializer"
	wamp_transport "wamp3go/transport"
)

func main() {
	session, e := wamp_transport.WebsocketJoin(
		"0.0.0.0:9999",
		&wamp_serializer.DefaultJSONSerializer,
	)
	if e != nil {
		panic("WebSocket Join")
	}

	type EchoPayload struct {
		Message string
	}

	registration, e := session.Register(
		"example.echo",
		&wamp.RegisterOptions{},
		func(callEvent wamp.CallEvent) wamp.ReplyEvent {
			payload := new(EchoPayload)
			callEvent.Payload(payload)
			replyEvent := wamp.NewReplyEvent(callEvent.ID(), payload)
			return replyEvent
		},
	)
	if e == nil {
		fmt.Printf("registration ID=%s\n", registration.ID)
	} else {
		panic("RegistrationError")
	}

	callEvent := wamp.NewCallEvent(
		&wamp.CallFeatures{"example.echo"},
		EchoPayload{"Hello, WAMP!"},
	)
	replyEvent := session.Call(callEvent)

	replyFeatures := replyEvent.Features()
	if replyFeatures.OK {
		replyPayload := new(EchoPayload)
		replyEvent.Payload(replyPayload)
		fmt.Printf("call(example.echo) %s\n", replyPayload)
	} else {
		e = wamp.ExtractError(replyEvent)
		fmt.Printf("call(example.echo) %s\n", e)
	}
}
