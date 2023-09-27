package main

import (
	"fmt"

	wamp "github.com/wamp3hub/wamp3go"
	wampSerializer "github.com/wamp3hub/wamp3go/serializer"
	wampTransport "github.com/wamp3hub/wamp3go/transport"
)

type EchoPayload struct {
	Message string
}

func echo(callEvent wamp.CallEvent) wamp.ReplyEvent {
	payload := new(EchoPayload)
	callEvent.Payload(payload)
	replyEvent := wamp.NewReplyEvent(callEvent.ID(), payload)
	return replyEvent
}

func main() {
	session, e := wampTransport.WebsocketJoin(
		"0.0.0.0:9999",
		&wampSerializer.DefaultJSONSerializer,
	)
	if e != nil {
		panic("WebSocket Join Error")
	}

	registration, e := session.Register("example.echo", &wamp.RegisterOptions{}, echo)
	if e == nil {
		fmt.Printf("registration ID=%s\n", registration.ID)
	} else {
		panic("RegisterError")
	}

	callEvent := wamp.NewCallEvent(&wamp.CallFeatures{"example.echo"}, EchoPayload{"Hello, WAMP!"})
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
