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

type EchoPayload struct {
	Message string
}

func echo(callEvent wamp.CallEvent) wamp.ReplyEvent {
	payload := new(EchoPayload)
	callEvent.Payload(payload)
	replyEvent := wamp.NewReplyEvent(callEvent, payload)
	return replyEvent
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
	bsession := createSession()

	registration, e := asession.Register("example.echo", &wamp.RegisterOptions{}, echo)
	if e == nil {
		fmt.Printf("registration ID=%s\n", registration.ID)
	} else {
		panic("RegisterError")
	}

	callEvent := wamp.NewCallEvent(&wamp.CallFeatures{"example.echo"}, EchoPayload{"Hello, WAMP!"})
	replyEvent := bsession.Call(callEvent)
	if replyEvent.Error() == nil {
		replyPayload := new(EchoPayload)
		replyEvent.Payload(replyPayload)
		fmt.Printf("call(example.echo) %s\n", replyPayload)
	} else {
		e = replyEvent.Error()
		fmt.Printf("call(example.echo) %s\n", e)
	}
}
