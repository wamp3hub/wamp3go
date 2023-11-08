package main

import (
	"fmt"
	"sync"

	wamp "github.com/wamp3hub/wamp3go"
	wampSerializers "github.com/wamp3hub/wamp3go/serializers"
	wampTransports "github.com/wamp3hub/wamp3go/transports"
)

func main() {
	wg := new(sync.WaitGroup)
	wg.Add(2)

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

	onEcho := func(publishEvent wamp.PublishEvent) {
		payload := ""
		publishEvent.Payload(&payload)
		fmt.Printf("new message %s\n", payload)
		wg.Done()
	}

	wamp.Subscribe(session, "example.echo", &wamp.SubscribeOptions{}, onEcho)

	wamp.Publish(session, &wamp.PublishFeatures{URI: "example.echo"}, "Hello, WAMP!")
	wamp.Publish(session, &wamp.PublishFeatures{URI: "example.echo"}, "How are you?")

	wg.Wait()
}
