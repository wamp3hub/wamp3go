package main

import (
	"fmt"
	"sync"

	wamp "github.com/wamp3hub/wamp3go"
	wampSerializer "github.com/wamp3hub/wamp3go/serializer"
	wampTransport "github.com/wamp3hub/wamp3go/transport"
)

type LoginPayload struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func createSession() *wamp.Session {
	session, e := wampTransport.WebsocketJoin(
		"0.0.0.0:8888",
		&wampSerializer.DefaultJSONSerializer,
		&LoginPayload{"test", "test"},
	)
	if e == nil {
		fmt.Printf("WAMP Join Success\n")
		return session
	}

	panic("WAMP Join Error")
}

func main() {
	wg := new(sync.WaitGroup)
	wg.Add(6)

	onEcho := func(publishEvent wamp.PublishEvent) {
		var payload string
		publishEvent.Payload(&payload)
		fmt.Printf("new message %s\n", payload)
		wg.Done()
	}

	asession := createSession()
	bsession := createSession()
	csession := createSession()
	dsession := createSession()

	asession.Subscribe("example.echo", &wamp.SubscribeOptions{}, onEcho)
	bsession.Subscribe("example.echo", &wamp.SubscribeOptions{}, onEcho)
	csession.Subscribe("example.echo", &wamp.SubscribeOptions{}, onEcho)

	publishEvent := wamp.NewPublishEvent(
		&wamp.PublishFeatures{"example.echo", nil, nil},
		"Hello, WAMP!",
	)
	dsession.Publish(publishEvent)

	publishEvent = wamp.NewPublishEvent(
		&wamp.PublishFeatures{"example.echo", nil, nil},
		"How are you?",
	)
	dsession.Publish(publishEvent)

	wg.Wait()
}
