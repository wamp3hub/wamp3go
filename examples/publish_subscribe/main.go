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

func main() {
	session, e := wampTransport.WebsocketJoin(
		"0.0.0.0:8888",
		&wampSerializer.DefaultJSONSerializer,
		&LoginPayload{"test", "test"},
	)
	if e == nil {
		fmt.Printf("WAMP Join Success\n")
	} else {
		panic("WAMP Join Error")
	}

	type EchoPayload struct {
		Message string
	}

	wg := new(sync.WaitGroup)
	wg.Add(1)

	subscription, e := session.Subscribe(
		"example.echo",
		&wamp.SubscribeOptions{},
		func(publishEvent wamp.PublishEvent) {
			payload := new(EchoPayload)
			publishEvent.Payload(payload)
			fmt.Printf("new message %s\n", payload.Message)
			wg.Done()
		},
	)
	if e == nil {
		fmt.Printf("subscription ID=%s\n", subscription.ID)
	} else {
		panic("SubscriptionError")
	}

	publishEvent := wamp.NewPublishEvent(
		&wamp.PublishFeatures{"example.echo", nil, nil},
		EchoPayload{"Hello, WAMP!"},
	)
	e = session.Publish(publishEvent)
	if e == nil {
		wg.Wait()
	} else {
		panic("Something went wrong")
	}
}
