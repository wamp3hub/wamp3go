package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

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
		&wampTransports.WebsocketJoinOptions{
			Secure: false,
			Address: "0.0.0.0:8888",
			Serializer: wampSerializers.DefaultSerializer,
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

	registration, e := wamp.Register(
		session,
		"net.example.greeting",
		&wamp.RegisterOptions{},
		func(name string, callEvent wamp.CallEvent) (string, error) {
			if len(name) == 0 {
				return "", errors.New("InvalidName")
			}
			result := "Hello, "+name+"!"
			return result, nil
		},
	)
	if e == nil {
		fmt.Printf("register success ID=%s\n", registration.ID)
	} else {
		panic("register error")
	}

	pendingResponse := wamp.Call[string](
		session,
		&wamp.CallFeatures{URI: "net.example.greeting"},
		"WAMP",
	)
	_, v, e := pendingResponse.Await()
	if e == nil {
		fmt.Printf("call(example.greeting) %s\n", v)
	} else {
		fmt.Printf("call(example.greeting) %s\n", e)
	}
}
