package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"sync"

	wamp "github.com/wamp3hub/wamp3go"
	wampSerializers "github.com/wamp3hub/wamp3go/serializers"
	wampTransports "github.com/wamp3hub/wamp3go/transports"
)

func main() {
	unixPath := flag.String("path", "", "unix socket path")
	flag.Parse()

	if len(*unixPath) == 0 {
		panic("unix socket path required")
	}

	wg := new(sync.WaitGroup)
	wg.Add(2)

	session, e := wampTransports.UnixJoin(
		&wampTransports.UnixJoinOptions{
			Path:       "/tmp/wamp.socket",
			Serializer: wampSerializers.DefaultSerializer,
			LoggingHandler: slog.NewTextHandler(
				os.Stdout,
				&slog.HandlerOptions{AddSource: false, Level: slog.LevelDebug},
			),
		},
	)
	if e == nil {
		fmt.Printf("WAMP Join Success session.ID=%s\n", session.ID())
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
