package main

import (
	"fmt"
	"log/slog"
	"os"
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
		&wampTransports.WebsocketJoinOptions{
			Secure:      false,
			Address:     "0.0.0.0:8888",
			Serializer:  wampSerializers.DefaultSerializer,
			Credentials: &LoginPayload{"test", "test"},
			LoggingHandler: slog.NewTextHandler(
				os.Stdout,
				&slog.HandlerOptions{AddSource: false, Level: slog.LevelInfo},
			),
		},
	)
	if e == nil {
		fmt.Printf("WAMP Join Success\n")
	} else {
		panic("WAMP Join Error")
	}

	wamp.Subscribe(
		session,
		"example.echo",
		&wamp.SubscribeOptions{},
		func(message string, publishEvent wamp.PublishEvent) {
			fmt.Printf("new message %s\n", message)
			wg.Done()
		},
	)

	wamp.Publish(session, &wamp.PublishFeatures{URI: "example.echo"}, "Hello, WAMP!")
	wamp.Publish(session, &wamp.PublishFeatures{URI: "example.echo"}, "How are you?")

	wg.Wait()
}
